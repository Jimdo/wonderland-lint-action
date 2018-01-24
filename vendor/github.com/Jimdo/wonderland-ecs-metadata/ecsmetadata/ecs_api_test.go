package ecsmetadata

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/golang/mock/gomock"

	mock_aws "github.com/Jimdo/wonderland-ecs-metadata/mock"
)

func TestECSAPI_GetService_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_aws.NewMockECSAPI(ctrl)
	m := &ECSAPI{
		ECS: client,
	}

	cluster := "foo"
	name := "bar"
	expected := &ecs.Service{
		ServiceName: aws.String(name),
	}

	client.EXPECT().DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: aws.StringSlice([]string{name}),
	}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{expected},
	}, nil)

	actual, err := m.GetService(cluster, name)
	if err != nil {
		t.Error("should not return an error when everything went well.")
	}
	if actual != expected {
		t.Error("should return the service from the ECS API.")
	}
}

func TestECSAPI_GetService_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_aws.NewMockECSAPI(ctrl)
	m := &ECSAPI{
		ECS: client,
	}

	cluster := "foo"
	name := "bar"

	client.EXPECT().DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: aws.StringSlice([]string{name}),
	}).Return(&ecs.DescribeServicesOutput{
		Services: nil,
	}, nil)

	service, err := m.GetService(cluster, name)
	if err != nil {
		t.Error("should not return an error when no service can be found.")
	}
	if service != nil {
		t.Error("should return nil when no service can be found.")
	}
}

func TestECSAPI_GetService_APIError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_aws.NewMockECSAPI(ctrl)
	m := &ECSAPI{
		ECS: client,
	}

	cluster := "foo"
	name := "bar"

	client.EXPECT().DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: aws.StringSlice([]string{name}),
	}).Return(nil, errors.New("This is an error"))

	service, err := m.GetService(cluster, name)
	if err == nil {
		t.Error("should return an error when the ECS API returns an error.")
	}
	if service != nil {
		t.Error("should return nil when there is an error.")
	}
}

func TestECSAPI_GetContainerInstance_withCluster(t *testing.T) {
	wantInstance := &ecs.ContainerInstance{
		ContainerInstanceArn: aws.String("arn:aws:ecs:eu-west-1:...:container-instance/abc123"),
	}

	ctrl := gomock.NewController(t)
	mockECS := mock_aws.NewMockECSAPI(ctrl)

	input := &ecs.DescribeContainerInstancesInput{
		Cluster: aws.String("cluster-a"),
		ContainerInstances: []*string{
			aws.String("arn:aws:ecs:eu-west-1:...:container-instance/abc123"),
		},
	}
	output := &ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []*ecs.ContainerInstance{
			wantInstance,
		},
	}
	mockECS.EXPECT().DescribeContainerInstances(gomock.Eq(input)).Times(1).Return(output, nil)

	api := &ECSAPI{
		ECS: mockECS,
	}

	gotInstance, err := api.GetContainerInstance("cluster-a", "arn:aws:ecs:eu-west-1:...:container-instance/abc123")
	if err != nil {
		t.Fatalf("Didn't expect error, but got: %s", err)
	}

	if aws.StringValue(gotInstance.ContainerInstanceArn) != aws.StringValue(wantInstance.ContainerInstanceArn) {
		t.Errorf(`api.GetContainerInstance("cluster-a", %q) = %q`, aws.StringValue(wantInstance.ContainerInstanceArn), aws.StringValue(gotInstance.ContainerInstanceArn))
	}
}
