package lconfig

import "fmt"

type PodInfo struct {
	Cluster           string `env:"POD_CLUSTER"`
	MonitoringCluster string `env:"POD_MONITORING_CLUSTER"`
	LoggingCluster    string `env:"POD_LOGGING_CLUSTER"`
	Namespace         string `env:"POD_NAMESPACE"`
	Name              string `env:"POD_NAME"`
}

func (info PodInfo) GetPromSearchLabels() string {
	return fmt.Sprintf(`cluster_name="%s",kubernetes_namespace="%s"`, info.MonitoringCluster, info.Namespace)
}
