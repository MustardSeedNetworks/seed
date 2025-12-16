// Package i18n provides internationalization support.
package i18n

// Message keys for authentication errors.
//
//nolint:gosec // G101: These are i18n message keys, not credentials.
const (
	ErrAuthInvalidCredentials = "errors.auth.invalidCredentials"
	ErrAuthAccountLocked      = "errors.auth.accountLocked"
	ErrAuthTooManyAttempts    = "errors.auth.tooManyAttempts"
	ErrAuthUnauthorized       = "errors.auth.unauthorized"
	ErrAuthNoToken            = "errors.auth.noToken"
	ErrAuthInvalidToken       = "errors.auth.invalidToken"
	ErrAuthExpiredToken       = "errors.auth.expiredToken"
	ErrAuthRefreshNotFound    = "errors.auth.refreshNotFound"
)

// Message keys for network errors.
const (
	ErrNetworkConnectionFailed  = "errors.network.connectionFailed"
	ErrNetworkError             = "errors.network.networkError"
	ErrNetworkTimeout           = "errors.network.timeout"
	ErrNetworkServerUnavailable = "errors.network.serverUnavailable"
)

// Message keys for API errors.
const (
	ErrAPIMethodNotAllowed   = "errors.api.methodNotAllowed"
	ErrAPIInvalidRequestBody = "errors.api.invalidRequestBody"
	ErrAPINotFound           = "errors.api.notFound"
	ErrAPIRateLimitExceeded  = "errors.api.rateLimitExceeded"
	ErrAPIInternalError      = "errors.api.internalError"
)

// Message keys for service errors.
const (
	ErrServiceNotAvailable   = "errors.service.notAvailable"
	ErrServiceNotEnabled     = "errors.service.notEnabled"
	ErrServiceAlreadyRunning = "errors.service.alreadyRunning"
	ErrServiceFailedToStart  = "errors.service.failedToStart"
	ErrServiceFailedToStop   = "errors.service.failedToStop"
)

// Message keys for config errors.
//
//nolint:gosec // G101: These are i18n message keys, not credentials.
const (
	ErrConfigFailedToSave        = "errors.config.failedToSave"
	ErrConfigFailedToLoad        = "errors.config.failedToLoad"
	ErrConfigInsecureCredentials = "errors.config.insecureCredentials"
)

// Message keys for validation errors.
const (
	ValLoginUsernameRequired = "validation.login.usernameRequired"
	ValLoginUsernameTooLong  = "validation.login.usernameTooLong"
	ValLoginPasswordRequired = "validation.login.passwordRequired"
	ValLoginPasswordTooLong  = "validation.login.passwordTooLong"

	ValThresholdWarningNonNegative      = "validation.threshold.warningNonNegative"
	ValThresholdCriticalNonNegative     = "validation.threshold.criticalNonNegative"
	ValThresholdWarningLessThanCritical = "validation.threshold.warningLessThanCritical"

	ValEndpointNameRequired    = "validation.endpoint.nameRequired"
	ValEndpointNameTooLong     = "validation.endpoint.nameTooLong"
	ValEndpointURLRequired     = "validation.endpoint.urlRequired"
	ValEndpointInvalidURL      = "validation.endpoint.invalidUrl"
	ValEndpointURLMustHaveHost = "validation.endpoint.urlMustHaveHost"
	ValEndpointPrivateURL      = "validation.endpoint.privateUrlNotAllowed"
	ValEndpointInvalidStatus   = "validation.endpoint.invalidStatus"

	ValHostRequired = "validation.host.required"
	ValHostInvalid  = "validation.host.invalid"

	ValAddressRequired = "validation.address.required"
	ValAddressInvalid  = "validation.address.invalid"

	ValInterfaceRequired        = "validation.interface.required"
	ValInterfaceTooLong         = "validation.interface.tooLong"
	ValInterfaceInvalid         = "validation.interface.invalid"
	ValInterfaceDefaultRequired = "validation.interface.defaultRequired"

	ValServerAddressRequired = "validation.server.addressRequired"
	ValServerHostnameTooLong = "validation.server.hostnameTooLong"
	ValServerInvalidAddress  = "validation.server.invalidAddress"

	ValPortInvalidRange = "validation.port.invalidRange"
	ValVLANInvalidRange = "validation.vlan.invalidRange"
	ValMTUInvalidRange  = "validation.mtu.invalidRange"

	ValNetmaskInvalidFormat = "validation.netmask.invalidFormat"
	ValNetmaskMustBeIPv4    = "validation.netmask.mustBeIPv4"
	ValNetmaskInvalidSubnet = "validation.netmask.invalidSubnet"

	ValIPModeInvalid  = "validation.ip.modeInvalid"
	ValProfileInvalid = "validation.profile.invalid"
)

// Message keys for API success responses.
const (
	APIStatusOK        = "api.status.ok"
	APIStatusSuccess   = "api.status.success"
	APIStatusUpdated   = "api.status.updated"
	APIStatusDeleted   = "api.status.deleted"
	APIStatusStarted   = "api.status.started"
	APIStatusStopped   = "api.status.stopped"
	APIStatusPaused    = "api.status.paused"
	APIStatusCompleted = "api.status.completed"
	APIStatusCreated   = "api.status.created"

	APIDiscoveryScanStarted     = "api.discovery.scanStarted"
	APIDiscoveryScanInProgress  = "api.discovery.scanInProgress"
	APIDiscoverySettingsUpdated = "api.discovery.settingsUpdated"
	APIDiscoverySubnetAdded     = "api.discovery.subnetAdded"
	APIDiscoverySubnetUpdated   = "api.discovery.subnetUpdated"
	APIDiscoverySubnetDeleted   = "api.discovery.subnetDeleted"
	APIDiscoveryProfileUpdated  = "api.discovery.profileUpdated"

	APITestsSettingsUpdated = "api.tests.settingsUpdated"
	APISpeedtestStarted     = "api.speedtest.started"

	APIIperfClientStarted = "api.iperf.clientStarted"
	APIIperfServerStarted = "api.iperf.serverStarted"
	APIIperfServerStopped = "api.iperf.serverStopped"

	APISettingsSNMPUpdated     = "api.settings.snmpUpdated"
	APISettingsWiFiUpdated     = "api.settings.wifiUpdated"
	APISettingsIPConfigUpdated = "api.settings.ipConfigUpdated"
	APISettingsMTUUpdated      = "api.settings.mtuUpdated"

	APIVLANCreated = "api.vlan.created"
	APIVLANDeleted = "api.vlan.deleted"

	APIDHCPServersCleared = "api.dhcp.serversCleared"

	APISurveySampleAdded      = "api.survey.sampleAdded"
	APISurveyFloorPlanUpdated = "api.survey.floorPlanUpdated"

	APIAuthLoggedOut = "api.auth.loggedOut"

	APIVulnerabilitiesScanStarted = "api.vulnerabilities.scanStarted"
	APIVulnerabilitiesNoData      = "api.vulnerabilities.noData"

	APIValidationFailed = "api.messages.validationFailed"
)
