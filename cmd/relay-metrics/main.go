package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/genproto/googleapis/api/metric"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	projectID = "nebula-235818"
	INTERVAL  = 11
)

var (
	Scheme = runtime.NewScheme()
)

func init() {
	builder := runtime.NewSchemeBuilder(
		scheme.AddToScheme,
		nebulav1.AddToScheme,
	)

	if err := builder.AddToScheme(Scheme); err != nil {
		panic(fmt.Sprintf("could not set up scheme for workflow run events: %+v", err))
	}
}

// createCustomMetric creates a custom metric specified by the metric type.
func createCustomMetric(ctx context.Context, mc *monitoring.MetricClient, w io.Writer, name, metricType, description string) (*metricpb.MetricDescriptor, error) {
	md := &metric.MetricDescriptor{
		Name:        name,
		DisplayName: name,
		Type:        metricType,
		MetricKind:  metric.MetricDescriptor_GAUGE,
		ValueType:   metric.MetricDescriptor_INT64,
		Unit:        "1",
		Description: description,
	}
	req := &monitoringpb.CreateMetricDescriptorRequest{
		Name:             "projects/" + projectID,
		MetricDescriptor: md,
	}
	m, err := mc.CreateMetricDescriptor(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("could not create custom metric: %v", err)
	}

	_, _ = fmt.Fprintf(w, "Created %s custom metric\n", m.GetName())
	return m, nil
}

// makeTimeSeriesValue constructs a a value for the custom metric created
func makeTimeSeriesValue(metricType, environment string, value int) *monitoringpb.TimeSeries {
	now := &timestamp.Timestamp{
		Seconds: time.Now().Unix(),
	}
	return &monitoringpb.TimeSeries{
		Metric: &metricpb.Metric{
			Type: metricType,
			Labels: map[string]string{
				"environment": environment,
			},
		},
		Resource: &monitoredres.MonitoredResource{
			Type: "global",
			Labels: map[string]string{
				"project_id": projectID,
			},
		},
		Points: []*monitoringpb.Point{{
			Interval: &monitoringpb.TimeInterval{
				EndTime: now,
			},
			Value: &monitoringpb.TypedValue{
				Value: &monitoringpb.TypedValue_Int64Value{
					Int64Value: int64(value),
				},
			},
		}},
	}
}

func writeTimeSeriesRequest(ctx context.Context, mc *monitoring.MetricClient, series []*monitoringpb.TimeSeries) error {
	req := &monitoringpb.CreateTimeSeriesRequest{
		Name:       "projects/" + projectID,
		TimeSeries: series,
	}
	log.Printf("writeTimeseriesRequest: %+v\n", req)

	err := mc.CreateTimeSeries(ctx, req)
	if err != nil {
		return fmt.Errorf("could not write time series value, %v ", err)
	}
	return nil
}

type workflowRunMetric struct {
	Name              string
	Status            string
	SecondsSinceStart float64
}

func getStatuses(ctx context.Context, c client.Client) (ret []*workflowRunMetric) {
	now := time.Now().UTC()
	wrs := &nebulav1.WorkflowRunList{}
	err := c.List(ctx, wrs)
	if err != nil {
		panic(err.Error())
	}
	for _, item := range wrs.Items {
		m := workflowRunMetric{
			Name:              item.Name,
			Status:            item.Status.Status,
			SecondsSinceStart: now.Sub(item.ObjectMeta.CreationTimestamp.UTC()).Seconds(),
		}
		fmt.Printf("%s is %s, %d\n", m.Name, m.Status, int(m.SecondsSinceStart))
		ret = append(ret, &m)
	}
	return
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func k8sClient(kubeconfig *string) client.Client {
	// Try homedir config
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		// Try in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	c, err := client.New(config, client.Options{
		Scheme: Scheme,
	})
	if err != nil {
		panic(err.Error())
	}

	return c
}

func metricsClient(ctx context.Context) *monitoring.MetricClient {
	mc, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		panic(err.Error())
	}

	return mc
}

func main() {
	ctx := context.Background()
	var err error
	var kubeconfig *string

	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	publish := flag.String("publish", "false", "Publish metrics to stackdriver")
	environment := flag.String("environment", "testing", "The environment being monitored for workflow runs")
	deleteMetrics := flag.String("delete", "false", "Delete the metric descriptors. Be very sure")

	flag.Parse()

	kc := k8sClient(kubeconfig)
	mc := metricsClient(ctx)

	statusName := "Pending Workflow Runs"
	oldestName := "Oldest Pending Workflow Run"
	statusType := "custom.googleapis.com/relay/workflowruns/status"
	oldestType := "custom.googleapis.com/relay/workflowruns/oldest_pending"

	if *deleteMetrics == "true" {
		err = deleteMetric(os.Stdout, "projects/"+projectID+"/metricDescriptors/"+statusType)
		if err != nil {
			panic(err.Error())
		}
		err = deleteMetric(os.Stdout, "projects/"+projectID+"/metricDescriptors/"+oldestType)
		if err != nil {
			panic(err.Error())
		}
		return
	}

	if publish != nil && *publish == "true" {
		_, err = createCustomMetric(ctx, mc, os.Stdout, statusName, statusType, "The number of Workflow Runs with a status of `pending`")
		if err != nil {
			panic(err.Error())
		}

		_, err = createCustomMetric(ctx, mc, os.Stdout, oldestName, oldestType, "The number of seconds since the oldest Workflow Runs with a status of `pending` was started")
		if err != nil {
			panic(err.Error())
		}
	}

	for {
		statuses := getStatuses(ctx, kc)
		count := 0
		oldest := 0.0
		for _, status := range statuses {
			if status.Status == "pending" {
				count += 1
				if status.SecondsSinceStart > oldest {
					oldest = status.SecondsSinceStart
					fmt.Printf("Found older: %d\n", int(oldest))
				}
			}
		}
		statusSeries := makeTimeSeriesValue(statusType, *environment, count)
		oldestSeries := makeTimeSeriesValue(oldestType, *environment, int(oldest))

		if publish != nil && *publish == "true" {
			err = writeTimeSeriesRequest(ctx, mc, []*monitoringpb.TimeSeries{statusSeries, oldestSeries})
			if err != nil {
				panic(err.Error())
			}
		} else {
			fmt.Println("not reporting")
		}

		// Stackdriver only wants points published every ten seconds or less
		time.Sleep(INTERVAL * time.Second)
	}
}

// deleteMetric deletes the given metric. name should be of the form
// "projects/PROJECT_ID/metricDescriptors/METRIC_TYPE".
func deleteMetric(w io.Writer, name string) error {
	ctx := context.Background()
	c, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return err
	}
	req := &monitoringpb.DeleteMetricDescriptorRequest{
		Name: name,
	}

	if err := c.DeleteMetricDescriptor(ctx, req); err != nil {
		return fmt.Errorf("could not delete metric: %v", err)
	}
	fmt.Fprintf(w, "Deleted metric: %q\n", name)
	return nil
}
