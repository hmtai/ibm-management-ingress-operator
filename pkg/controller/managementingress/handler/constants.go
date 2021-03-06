//
// Copyright 2020 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package handler

const (
	AppName                   string = "management-ingress"
	ServiceName               string = "icp-management-ingress"
	ConfigName                string = "management-ingress-config"
	IAMTokenService           string = "iam-token-service"
	ServiceAccountName        string = "management-ingress"
	CertName                  string = "management-ingress-cert"
	TLSSecretName             string = "icp-management-ingress-tls-secret"
	RouteName                 string = "cp-console"
	RouteCert                 string = "route-cert"
	RouteSecret               string = "route-tls-secret"
	csPriorityClassName       string = "cs-priority-class"
	ConfigUpdateAnnotationKey string = "management-ingress.operator.k8s.io/config-updated"
	SCCName                   string = "management-ingress-scc"
	BindInfoConfigMap         string = "management-ingress-info"

	// Dependency from IAM service
	PlatformAuthConfigmap string = "platform-auth-idp"
	PlatformAuthSecret    string = "platform-oidc-credentials"

	// Product info required by metering
	ProductName    string = "IBM Cloud Platform Common Services"
	ProductID      string = "068a62892a1e4db39641342e592daa25"
	ProductVersion string = "3.3.0"
	ProductMetric  string = "FREE"
)
