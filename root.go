/*
Copyright © 2020 Krunal Patel <krpatel19@outlook.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package memory_used_percent

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/monitoredresource"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/spf13/cobra"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

var (
	memUsedPerc = stats.Float64("sys_mem_used", "Memory Usage In Percent", "%")
	usedPerc    float64
	prefix      string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sys_mem_used_percent",
	Short: "Send GCP Custom Monitoring Metric for System Memory Used Percent Every Minute Interval",
	Long: `Run "sys_mem_used_percent -h" command for how to execute run this application. 
You can run this application through systemd unit as well.No cronjob required.
This is application should run inside GCP Virtual Machines for accurate Results.
The Same metrics can be used to autoscale MIG(stateless) using autoscaler custom monitoring metrics option.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		// Register the view. It is imperative that this step exists,
		// otherwise recorded metrics will be dropped and never exported.
		// We collect GAUGE Measurement
		v := &view.View{
			Name:        "sys_mem_used_percent",
			Measure:     memUsedPerc,
			Description: "Memory Usage In Percent",
			Aggregation: view.LastValue(),
		}
		if err := view.Register(v); err != nil {
			log.Fatalf("Failed to register the view: %v", err)
		}

		projectID, err := metadata.ProjectID()
		if err != nil {
			log.Println("Error getting Project ID:", err)
			projectID = "unknown"
		}

		// Enable OpenCensus exporters to export metrics
		// to Stackdriver Monitoring.
		// Exporters use Application Default Credentials to authenticate.
		// See https://developers.google.com/identity/protocols/application-default-credentials
		// for more details.
		exporter, err := stackdriver.NewExporter(stackdriver.Options{
			ProjectID:               projectID,
			MetricPrefix:            prefix,
			MonitoredResource:       monitoredresource.Autodetect(),
			DefaultMonitoringLabels: &stackdriver.Labels{},
			ReportingInterval:       60 * time.Second,
		})
		if err != nil {
			log.Fatal(err)
		}
		// Flush must be called before main() exits to ensure metrics are recorded.
		defer exporter.Flush()

		if err := exporter.StartMetricsExporter(); err != nil {
			log.Fatalf("Error starting metric exporter: %v", err)
		}
		defer exporter.StopMetricsExporter()

		for {
			// Get system memory usage
			memory, err := memory.Get()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}
			usedPerc = float64(memory.Used) / float64(memory.Total) * 100
			//fmt.Printf("Memory Usage %f\n", usedPerc) // For Debugging purpose
			stats.Record(ctx, memUsedPerc.M(usedPerc))
			//fmt.Println("Done recording metrics")
			time.Sleep(60 * time.Second)
		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&prefix, "metric_prefix", "p", "custom", `Usage: memory_used_percent [prefix] where prefix is Metricprefix.
Your GCP custom monitoring metric name should look like e.g. if prefix is set to tester
then your metric name will be custom.googleapis.com/opencensus/tester/sys_mem_used_percent.
Default is "custom" so your metric name will be visible as 
custom.googleapis.com/opencensus/custom/sys_mem_used_percent in stackdriver monitoring Console.
Run "sys_mem_used_percent -h" command for how to execute run this application.
You can run this application through systemd unit as well.No cronjob required.
This is application should run inside GCP Virtual Machines for accurate Results.
The Same metrics can be used to autoscale MIG(stateless) using autoscaler custom monitoring metrics option.`)
	rootCmd.MarkFlagRequired("metric_prefix")
}
