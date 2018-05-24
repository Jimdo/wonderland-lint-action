package v2

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/Jimdo/wonderland-crons/mock"
)

func TestExecutionTriggerHandler_Notification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mock.NewMockCronService(ctrl)
	service.EXPECT().TriggerExecutionByRuleARN(gomock.Any())

	body := `{
		"Type" : "Notification",
		"MessageId" : "230bd4e4-bfe9-52e6-b9fe-8692dff2da86",
		"TopicArn" : "arn:aws:sns:eu-west-1:062052581233:side-test",
		"Message" : "{\"version\":\"0\",\"id\":\"dc031b75-7deb-0b25-4b88-044a0662afb1\",\"detail-type\":\"Scheduled Event\",\"source\":\"aws.events\",\"account\":\"062052581233\",\"time\":\"2017-11-03T12:15:02Z\",\"region\":\"eu-west-1\",\"resources\":[\"arn:aws:events:eu-west-1:062052581233:rule/side-test-cron\"],\"detail\":{}}",
		"Timestamp" : "2017-11-03T12:15:23.973Z",
		"SignatureVersion" : "1",
		"Signature" : "HfJ6cNuqOS2UXRNRVjKYqPMnMGmi3GBDEqQES85MeekrGXZcHGGNIPFxfq6cocvVtGqTuLcLTHnIFV4zLq/samkfsyB1+nhIyUN0qafJ2YWng4dDTC/IYzbBWhgBQGdkuvp0XEyPeKnFip7LV3W4KA4hD+favRG5MHWbQLEvJjA8BMpdWn6x7vGLg/XuNRRdYH7USuX0TwTsfkyr2Afbmz6Qi2SlmweL6vzalRjUcf189gvXnHzSyE8iBptVwWYtOYGt8v9uhw8II8N985OqmtT6u7ndbECLYZUcWUY2xekSRo9W5RUfFGa3K8/+5y4NHx8EDPulUReVxKHk6VZRbg==",
		"SigningCertURL" : "https://sns.eu-west-1.amazonaws.com/SimpleNotificationService-433026a4050d206028891664da859041.pem",
		"UnsubscribeURL" : "https://sns.eu-west-1.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:eu-west-1:062052581233:side-test:f4be8ff3-8be3-402d-bb42-db179794d6d8"
	  }`
	api := &API{
		config: &Config{
			Service: service,
		},
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/bla", bytes.NewBufferString(body))
	req.Header.Add("x-amz-sns-message-type", "Notification")
	api.ExecutionTriggerHandler(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestExecutionTriggerHandler_SubscriptionConfirmation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedSubscribeURL := "http://www.example.com/subscribe"
	expectedReq, _ := http.NewRequest(http.MethodGet, expectedSubscribeURL, nil)
	resp := &http.Response{StatusCode: http.StatusOK}
	hc := mock.NewMockHTTPClient(ctrl)
	hc.EXPECT().Do(expectedReq).Return(resp, nil)

	body := fmt.Sprintf(`{"SubscribeURL" : "%s"}`, expectedSubscribeURL)
	api := &API{
		config: &Config{},
		hc:     hc,
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/bla", bytes.NewBufferString(body))
	req.Header.Add("x-amz-sns-message-type", "SubscriptionConfirmation")
	api.ExecutionTriggerHandler(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
