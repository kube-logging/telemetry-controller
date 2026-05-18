// Copyright © 2026 Kube logging authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exporter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
)

func TestGenerateAWSS3Exporters(t *testing.T) {
	tests := []struct {
		name              string
		resourceRelations components.ResourceRelations
		expectedResult    map[string]any
	}{
		{
			name: "Basic bucket + region",
			resourceRelations: components.ResourceRelations{
				Tenants: []v1alpha1.Tenant{
					{
						ObjectMeta: metav1.ObjectMeta{Name: testTenantName},
					},
				},
				Subscriptions: map[v1alpha1.NamespacedName]v1alpha1.Subscription{
					{Name: "default", Namespace: "default"}: {
						ObjectMeta: metav1.ObjectMeta{Name: "sub1", Namespace: "default"},
						Spec: v1alpha1.SubscriptionSpec{
							Outputs: []v1alpha1.NamespacedName{
								{Name: "s3-output1", Namespace: "default"},
							},
						},
					},
				},
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{Name: "s3-output1", Namespace: "default"},
							Spec: v1alpha1.OutputSpec{
								AWSS3: &v1alpha1.AWSS3{
									S3Uploader: &v1alpha1.S3UploaderConfig{
										Region:   "us-east-1",
										S3Bucket: "telemetry-bucket",
									},
								},
							},
						},
					},
				},
				TenantSubscriptionMap: map[string][]v1alpha1.NamespacedName{
					testTenantName: {{Name: "sub1", Namespace: "default"}},
				},
				SubscriptionOutputMap: map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{
					{Name: "sub1", Namespace: "default"}: {
						{Name: "s3-output1", Namespace: "default"},
					},
				},
			},
			expectedResult: map[string]any{
				"awss3/default_s3-output1": map[string]any{
					"s3uploader": map[string]any{
						"region":    "us-east-1",
						"s3_bucket": "telemetry-bucket",
					},
					"sending_queue": map[string]any{
						"enabled":    true,
						"queue_size": float64(1000),
					},
					"retry_on_failure": map[string]any{
						"enabled":          true,
						"max_elapsed_time": float64(0),
					},
				},
			},
		},
		{
			name: "With persistence enabled",
			resourceRelations: components.ResourceRelations{
				Tenants: []v1alpha1.Tenant{
					{
						ObjectMeta: metav1.ObjectMeta{Name: testTenantName},
						Spec: v1alpha1.TenantSpec{
							PersistenceConfig: v1alpha1.PersistenceConfig{
								EnableFileStorage: true,
							},
						},
					},
				},
				Subscriptions: map[v1alpha1.NamespacedName]v1alpha1.Subscription{
					{Name: "default", Namespace: "default"}: {
						ObjectMeta: metav1.ObjectMeta{Name: "sub2", Namespace: "default"},
						Spec: v1alpha1.SubscriptionSpec{
							Outputs: []v1alpha1.NamespacedName{
								{Name: "s3-output2", Namespace: "default"},
							},
						},
					},
				},
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{Name: "s3-output2", Namespace: "default"},
							Spec: v1alpha1.OutputSpec{
								AWSS3: &v1alpha1.AWSS3{
									S3Uploader: &v1alpha1.S3UploaderConfig{
										Region:   "eu-west-1",
										S3Bucket: "tenant-bucket",
									},
								},
							},
						},
					},
				},
				TenantSubscriptionMap: map[string][]v1alpha1.NamespacedName{
					testTenantName: {{Name: "sub2", Namespace: "default"}},
				},
				SubscriptionOutputMap: map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{
					{Name: "sub2", Namespace: "default"}: {
						{Name: "s3-output2", Namespace: "default"},
					},
				},
			},
			expectedResult: map[string]any{
				"awss3/default_s3-output2": map[string]any{
					"s3uploader": map[string]any{
						"region":    "eu-west-1",
						"s3_bucket": "tenant-bucket",
					},
					"sending_queue": map[string]any{
						"enabled":    true,
						"queue_size": float64(1000),
						"storage":    "file_storage/" + testTenantName,
					},
					"retry_on_failure": map[string]any{
						"enabled":          true,
						"max_elapsed_time": float64(0),
					},
				},
			},
		},
		{
			name: "All fields set",
			resourceRelations: components.ResourceRelations{
				Tenants: []v1alpha1.Tenant{
					{
						ObjectMeta: metav1.ObjectMeta{Name: testTenantName},
					},
				},
				Subscriptions: map[v1alpha1.NamespacedName]v1alpha1.Subscription{
					{Name: "default", Namespace: "default"}: {
						ObjectMeta: metav1.ObjectMeta{Name: "sub3", Namespace: "default"},
						Spec: v1alpha1.SubscriptionSpec{
							Outputs: []v1alpha1.NamespacedName{
								{Name: "s3-output3", Namespace: "prod"},
							},
						},
					},
				},
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{Name: "s3-output3", Namespace: "prod"},
							Spec: v1alpha1.OutputSpec{
								AWSS3: &v1alpha1.AWSS3{
									MarshalerName:         new("otlp_proto"),
									EncodingFileExtension: ".bin",
									S3Uploader: &v1alpha1.S3UploaderConfig{
										Region:              "us-west-2",
										S3Bucket:            "logs-bucket",
										S3BasePrefix:        "telemetry",
										S3Prefix:            "logs",
										S3PartitionFormat:   "year=%Y/month=%m/day=%d",
										S3PartitionTimezone: "UTC",
										FilePrefix:          "otel-",
										Endpoint:            "https://s3.example.com",
										RoleArn:             "arn:aws:iam::123456789012:role/otel",
										S3ForcePathStyle:    true,
										DisableSSL:          false,
										ACL:                 "bucket-owner-full-control",
										StorageClass:        "STANDARD_IA",
										Compression:         "gzip",
										RetryMode:           "adaptive",
										RetryMaxAttempts:    5,
										RetryMaxBackoff:     30 * time.Second,
										UniqueKeyFuncName:   "uuidv7",
									},
									ResourceAttrsToS3: &v1alpha1.ResourceAttrsToS3{
										S3Bucket: "telemetry.bucket",
										S3Prefix: "telemetry.prefix",
									},
									QueueConfig: &v1alpha1.QueueSettings{
										NumConsumers: new(8),
										QueueSize:    new(2048),
									},
								},
							},
						},
					},
				},
				TenantSubscriptionMap: map[string][]v1alpha1.NamespacedName{
					testTenantName: {{Name: "sub3", Namespace: "default"}},
				},
				SubscriptionOutputMap: map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{
					{Name: "sub3", Namespace: "default"}: {
						{Name: "s3-output3", Namespace: "prod"},
					},
				},
			},
			expectedResult: map[string]any{
				"awss3/prod_s3-output3": map[string]any{
					"marshaler":               "otlp_proto",
					"encoding_file_extension": ".bin",
					"s3uploader": map[string]any{
						"region":                "us-west-2",
						"s3_bucket":             "logs-bucket",
						"s3_base_prefix":        "telemetry",
						"s3_prefix":             "logs",
						"s3_partition_format":   "year=%Y/month=%m/day=%d",
						"s3_partition_timezone": "UTC",
						"file_prefix":           "otel-",
						"endpoint":              "https://s3.example.com",
						"role_arn":              "arn:aws:iam::123456789012:role/otel",
						"s3_force_path_style":   true,
						"acl":                   "bucket-owner-full-control",
						"storage_class":         "STANDARD_IA",
						"compression":           "gzip",
						"retry_mode":            "adaptive",
						"retry_max_attempts":    float64(5),
						"retry_max_backoff":     float64(30 * time.Second),
						"unique_key_func_name":  "uuidv7",
					},
					"resource_attrs_to_s3": map[string]any{
						"s3_bucket": "telemetry.bucket",
						"s3_prefix": "telemetry.prefix",
					},
					"sending_queue": map[string]any{
						"enabled":       true,
						"num_consumers": float64(8),
						"queue_size":    float64(2048),
					},
					"retry_on_failure": map[string]any{
						"enabled":          true,
						"max_elapsed_time": float64(0),
					},
				},
			},
		},
		{
			name: "No S3 output",
			resourceRelations: components.ResourceRelations{
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{Name: "non-s3", Namespace: "default"},
							Spec: v1alpha1.OutputSpec{
								File: &v1alpha1.File{Path: "/tmp/out.json"},
							},
						},
					},
				},
			},
			expectedResult: map[string]any{},
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, ttp.expectedResult, GenerateAWSS3Exporters(context.TODO(), ttp.resourceRelations))
		})
	}
}
