package logger

import (
	"os"
	"runtime"
)

type Config struct {
	Level      Level             `json:"level"       yaml:"level"`
	Format     string            `json:"format"      yaml:"format"` // json, text
	Output     string            `json:"output"      yaml:"output"` // stdout, stderr, file
	FilePath   string            `json:"file_path"   yaml:"file_path"`
	MaxSize    int               `json:"max_size"    yaml:"max_size"` // MB
	MaxBackups int               `json:"max_backups" yaml:"max_backups"`
	MaxAge     int               `json:"max_age"     yaml:"max_age"` // days
	Compress   bool              `json:"compress"    yaml:"compress"`
	Fields     map[string]string `json:"fields"      yaml:"fields"` // static fields for k8s/docker
}

func GetDefaultFields() Fields {
	hostname, _ := os.Hostname()

	fields := Fields{
		"hostname":   hostname,
		"pid":        os.Getpid(),
		"go_version": runtime.Version(),
	}

	// Kubernetes fields
	if namespace := os.Getenv("KUBERNETES_NAMESPACE"); namespace != "" {
		fields["k8s_namespace"] = namespace
	}
	if podName := os.Getenv("KUBERNETES_POD_NAME"); podName != "" {
		fields["k8s_pod"] = podName
	}
	if nodeName := os.Getenv("KUBERNETES_NODE_NAME"); nodeName != "" {
		fields["k8s_node"] = nodeName
	}
	if serviceName := os.Getenv("KUBERNETES_SERVICE_NAME"); serviceName != "" {
		fields["k8s_service"] = serviceName
	}

	// Docker fields
	if containerID := os.Getenv("HOSTNAME"); containerID != "" {
		fields["container_id"] = containerID
	}
	if imageName := os.Getenv("DOCKER_IMAGE"); imageName != "" {
		fields["docker_image"] = imageName
	}

	// AWS CloudWatch fields
	if region := os.Getenv("AWS_REGION"); region != "" {
		fields["aws_region"] = region
	}
	if logGroup := os.Getenv("AWS_LOG_GROUP"); logGroup != "" {
		fields["aws_log_group"] = logGroup
	}
	if logStream := os.Getenv("AWS_LOG_STREAM"); logStream != "" {
		fields["aws_log_stream"] = logStream
	}

	// Application fields
	if appName := os.Getenv("APP_NAME"); appName != "" {
		fields["app_name"] = appName
	}
	if appVersion := os.Getenv("APP_VERSION"); appVersion != "" {
		fields["app_version"] = appVersion
	}
	if env := os.Getenv("APP_ENV"); env != "" {
		fields["environment"] = env
	}

	return fields
}

func NewDefaultConfig() *Config {
	config := &Config{
		Level:      LevelInfo,
		Format:     "console", // Default to console for development
		Output:     "stdout",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
		Fields:     make(map[string]string),
	}

	// Convert default fields to string map
	defaultFields := GetDefaultFields()
	for k, v := range defaultFields {
		if str, ok := v.(string); ok {
			config.Fields[k] = str
		}
	}

	return config
}
