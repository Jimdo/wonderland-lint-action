package validation

import (
	"testing"

	"github.com/Jimdo/wonderland-cron/cron"
)

func TestValidateCapacityDescription_Valid(t *testing.T) {
	valid := []cron.CapacityDescription{
		{Memory: "XS", CPU: "XS"},
		{Memory: "S", CPU: "L"},
		{Memory: "M", CPU: "XL"},
		{Memory: "XXL", CPU: "2XL"},
		{Memory: "XXXL", CPU: "3XL"},
		{Memory: "1023", CPU: "1028"},
		{Memory: "S", CPU: "1023"},
		{Memory: "1023", CPU: "XL"},
	}

	v := &containerDescription{}
	for _, capacity := range valid {
		if err := v.validateCapacityDescription(&capacity); err != nil {
			t.Errorf("%+v should be a valid capacity description: %s", capacity, err)
		}
	}
}

func TestValidateCapacityDescription_Invalid(t *testing.T) {
	invalid := []cron.CapacityDescription{
		{Memory: "A", CPU: "XS"},
		{Memory: "XS", CPU: "X"},
		{Memory: "XXL", CPU: "XXXXL"},
		{Memory: "4XL", CPU: "XXXXL"},
		{Memory: "1", CPU: "XL"},
		{Memory: "L", CPU: "2"},
		{Memory: "1", CPU: "2"},
		{Memory: "10000", CPU: "XL"},
		{Memory: "L", CPU: "2096"},
		{Memory: "10000", CPU: "2096"},
	}

	v := &containerDescription{}
	for _, capacity := range invalid {
		if err := v.validateCapacityDescription(&capacity); err == nil {
			t.Errorf("%+v should not be a valid capacity description", capacity)
		}
	}
}
