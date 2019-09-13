// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

const (
	// MeasurementsHeadCluster is the cluster row identifier in the measurement file.
	MeasurementsHeadCluster = "cluster"

	// MeasurementsHeadProvider is the provider row identifier in the measurement file.
	MeasurementsHeadProvider = "provider"

	// MeasurementsHeadSeed is the seed row identifier in the measurement file.
	MeasurementsHeadSeed = "seed"

	// MeasurementsHeadTimestamp is the timestamp row identifier in the measurement file.
	MeasurementsHeadTimestamp = "timestamp"

	// MeasurementsHeadStatusCode is the status code row identifier in the measurement file.
	MeasurementsHeadStatusCode = "status_code"

	// MeasurementsHeadResponseTime is the response time row identifier in the measurement file.
	MeasurementsHeadResponseTime = "response_time_ms"

	// ReportOutputFormatText is the identifier for the report output format text.
	ReportOutputFormatText = "text"

	// ReportOutputFormatJSON is the identifier for the report output format json.
	ReportOutputFormatJSON = "json"

	// CliFlagLogLevel is the cli flag to specify the log level.
	CliFlagLogLevel = "log-level"

	// CliFlagReportOutput is the cli flag which passes the report file destination.
	CliFlagReportOutput = "report"

	// CliFlagReportFormat is the cli flag which passes the report output format.
	CliFlagReportFormat = "format"

	// CliFlagHelpTextReportFile is the help text for the cli flag which passes the report file destination.
	CliFlagHelpTextReportFile = "path to the report file"

	// CliFlagHelpTextReportFormat is the help text for the cli flag which passes the report output format.
	CliFlagHelpTextReportFormat = "output format of the report: text|json"

	// CliFlagHelpLogLevel is the help text for the cli flag which specify the log level.
	CliFlagHelpLogLevel = "log level: error|info|debug"

	// LogDebugAddPrefix is a prefix for controller add operations debug log outputs.
	LogDebugAddPrefix = "[ADD]"

	// LogDebugUpdatePrefix is a prefix for controller update operations debug log outputs.
	LogDebugUpdatePrefix = "[UPDATE]"

	// DefaultLogLevel define the default log level.
	DefaultLogLevel = "info"

	// RequestTimeOut is the timeout for a health check to the ApiServer.
	RequestTimeOut int = 5000
)
