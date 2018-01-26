package ecsmetadata

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/golang/mock/gomock"

	"github.com/Jimdo/wonderland-ecs-metadata/mock"
)

func TestFallback_GetContainerInstances_FirstSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	first := mock.NewMockMetadata(ctrl)
	second := mock.NewMockMetadata(ctrl)

	f := &Fallback{
		Primary:   first,
		Secondary: second,
	}

	expected := []*ecs.ContainerInstance{{
		ContainerInstanceArn: aws.String("arn:..."),
	}}
	cluster := "foo"

	first.EXPECT().GetContainerInstances(cluster).Return(expected, nil)
	second.EXPECT().GetContainerInstances(cluster).Times(0)

	actual, err := f.GetContainerInstances(cluster)
	if err != nil {
		t.Errorf("should not return an error when everything worked (err: %s)", err)
	}
	if len(actual) != len(expected) || actual[0] != expected[0] {
		t.Errorf("should return the same result as the wrapped metadata provider. (%#v != %#v)", actual, expected)
	}
}

func TestFallback_GetContainerInstances_SecondSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	first := mock.NewMockMetadata(ctrl)
	second := mock.NewMockMetadata(ctrl)

	f := &Fallback{
		Primary:   first,
		Secondary: second,
	}

	expected := []*ecs.ContainerInstance{{
		ContainerInstanceArn: aws.String("arn:..."),
	}}
	cluster := "foo"

	first.EXPECT().GetContainerInstances(cluster).Return(nil, errors.New("This is an error"))
	second.EXPECT().GetContainerInstances(cluster).Return(expected, nil)

	actual, err := f.GetContainerInstances(cluster)
	if err != nil {
		t.Errorf("should not return an error when fallback worked (err: %s)", err)
	}
	if len(actual) != len(expected) || actual[0] != expected[0] {
		t.Errorf("should return the same result as the wrapped metadata provider. (%#v != %#v)", actual, expected)
	}
}

func TestFallback_GetContainerInstances_Failure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	first := mock.NewMockMetadata(ctrl)
	second := mock.NewMockMetadata(ctrl)

	f := &Fallback{
		Primary:   first,
		Secondary: second,
	}

	cluster := "foo"

	first.EXPECT().GetContainerInstances(cluster).Return(nil, errors.New("This is an error"))
	second.EXPECT().GetContainerInstances(cluster).Return(nil, errors.New("This is another error"))

	actual, err := f.GetContainerInstances(cluster)
	if err == nil {
		t.Error("should return an error when even fallback did not work")
	}
	if actual != nil {
		t.Error("should not return a result")
	}
}
