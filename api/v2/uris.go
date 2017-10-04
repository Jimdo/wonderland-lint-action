package v2

import "fmt"

type URIGenerator struct {
	LogzioAccountID string
	LogzioURL       string
}

func (u *URIGenerator) CronLogsHTML(cron string) string {
	return fmt.Sprintf("%s/#/dashboard/kibana?kibanaRoute=%%2Fdiscover%%3F_g%%3D()%%26_a%%3D(columns%%3A!(message)%%2Cindex%%3A%%255BlogzioCustomerIndex%%255DYYMMDD%%2Cinterval%%3Aauto%%2Cquery%%3A(query_string%%3A(analyze_wildcard%%3A!t%%2Cquery%%3A%%2527wonderland.cron%%3A%s%%2527))%%2Csort%%3A!(%%2527%%40timestamp%%2527%%2Cdesc))&switchToAccountId=%s", u.LogzioURL, cron, u.LogzioAccountID)
}
