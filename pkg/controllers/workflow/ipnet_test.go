package workflow

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIPNetExclude(t *testing.T) {
	tests := []struct {
		Name     string
		CIDR     string
		Exclude  []string
		Expected []string
	}{
		{
			Name: "small",
			CIDR: "10.1.2.0/28",
			Exclude: []string{
				"10.1.2.4",
				"10.1.2.1",
			},
			Expected: []string{
				"10.1.2.0/32",
				"10.1.2.2/31",
				"10.1.2.5/32",
				"10.1.2.6/31",
				"10.1.2.8/29",
			},
		},
		{
			Name: "large",
			CIDR: "10.0.0.0/8",
			Exclude: []string{
				"10.193.34.1",
				"10.193.2.1",
			},
			Expected: []string{
				"10.193.2.0/32",
				"10.193.2.2/31",
				"10.193.2.4/30",
				"10.193.2.8/29",
				"10.193.2.16/28",
				"10.193.2.32/27",
				"10.193.2.64/26",
				"10.193.2.128/25",
				"10.193.3.0/24",
				"10.193.0.0/23",
				"10.193.4.0/22",
				"10.193.8.0/21",
				"10.193.16.0/20",
				"10.193.34.0/32",
				"10.193.34.2/31",
				"10.193.34.4/30",
				"10.193.34.8/29",
				"10.193.34.16/28",
				"10.193.34.32/27",
				"10.193.34.64/26",
				"10.193.34.128/25",
				"10.193.35.0/24",
				"10.193.32.0/23",
				"10.193.36.0/22",
				"10.193.40.0/21",
				"10.193.48.0/20",
				"10.193.64.0/18",
				"10.193.128.0/17",
				"10.192.0.0/16",
				"10.194.0.0/15",
				"10.196.0.0/14",
				"10.200.0.0/13",
				"10.208.0.0/12",
				"10.224.0.0/11",
				"10.128.0.0/10",
				"10.0.0.0/9",
			},
		},
		{
			Name: "no overlap",
			CIDR: "10.0.0.0/8",
			Exclude: []string{
				"192.168.1.1",
			},
			Expected: []string{
				"10.0.0.0/8",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			_, nw, err := net.ParseCIDR(test.CIDR)
			require.NoError(t, err)

			ips := make([]net.IP, len(test.Exclude))
			for i, e := range test.Exclude {
				ips[i] = net.ParseIP(e)
			}

			res := ipNetExclude(nw, ips)
			actual := make([]string, len(res))
			for i, r := range res {
				actual[i] = r.String()
			}

			require.Equal(t, test.Expected, actual)
		})
	}
}
