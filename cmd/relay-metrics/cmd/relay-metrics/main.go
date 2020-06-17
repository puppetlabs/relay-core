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

	nebulav1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	metricName = "Test Pending Workflow Runs"
	metricType = "custom.googleapis.com/relay/workflowruns/status"
	projectID  = "nebula-235818"
	INTERVAL   = 10
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
func createCustomMetric(ctx context.Context, mc *monitoring.MetricClient, w io.Writer, name, description string) (*metricpb.MetricDescriptor, error) {
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

// writeTimeSeriesValue writes a value for the custom metric created
func makeTimeSeriesValue(environment string, value int) *monitoringpb.TimeSeries {
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

func writeTimeSeriesRequest(ctx context.Context, mc *monitoring.MetricClient, series *monitoringpb.TimeSeries) error {
	req := &monitoringpb.CreateTimeSeriesRequest{
		Name:       "projects/" + projectID,
		TimeSeries: []*monitoringpb.TimeSeries{series},
	}
	log.Printf("writeTimeseriesRequest: %+v\n", req)

	err := mc.CreateTimeSeries(ctx, req)
	if err != nil {
		return fmt.Errorf("could not write time series value, %v ", err)
	}
	return nil
}

type workflowRunMetric struct {
	Name   string
	Status string
}

func getStatuses(ctx context.Context, c client.Client) (ret []*workflowRunMetric) {
	wrs := &nebulav1.WorkflowRunList{}
	err := c.List(ctx, wrs)
	if err != nil {
		panic(err.Error())
	}
	for _, item := range wrs.Items {
		fmt.Printf("%s is %s\n", item.Name, item.Status.Status)
		ret = append(ret, &workflowRunMetric{
			Name:   item.Name,
			Status: item.Status.Status,
		})
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

	flag.Parse()

	kc := k8sClient(kubeconfig)
	mc := metricsClient(ctx)

	if publish != nil && *publish == "true" {
		_, err = createCustomMetric(ctx, mc, os.Stdout, metricName, "The number of Workflow Runs with a status of `pending`")
		if err != nil {
			panic(err.Error())
		}
	}

	for {
		statuses := getStatuses(ctx, kc)
		count := 0
		for _, status := range statuses {
			if status.Status == "pending" {
				count += 1
			}
		}
		series := makeTimeSeriesValue(*environment, count)

		if publish != nil && *publish == "true" {
			err := writeTimeSeriesRequest(ctx, mc, series)
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
