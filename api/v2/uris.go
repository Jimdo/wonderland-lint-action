package v2

import "fmt"

type URIGenerator struct {
	LogzioAccountID string
	LogzioURL       string
}

func (u *URIGenerator) CronLogsHTML(cron string) string {
	return fmt.Sprintf("%s/#/dashboard/kibana/discover?_a=(columns%%3A!(message)%%2Cindex%%3A'logzioCustomerIndex*'%%2Cinterval%%3Aauto%%2Cquery%%3A(language%%3Alucene%%2Cquery%%3A'wonderland.cron%%3A%s')%%2Csort%%3A!('%%40timestamp'%%2Cdesc))&_g=()&switchToAccountId=%s", u.LogzioURL, cron, u.LogzioAccountID)
}
