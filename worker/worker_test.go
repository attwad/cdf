package worker

import (
	"fmt"
	"testing"

	"github.com/attwad/cdf/pick"
)

type fakePicker struct {
	pick.Picker
}

func (p *fakePicker) ScheduleRandom() error {
	return nil
}

type fakeBroker struct {
	balance      int
	balanceError error
}

func (b *fakeBroker) GetBalance() (int, error) {
	return b.balance, b.balanceError
}

func (b *fakeBroker) ChangeBalance(delta int) error {
	b.balance -= delta
	return nil
}

func TestMaybeSchedule(t *testing.T) {
	fb := &fakeBroker{balance: 50}
	w := Worker{
		broker:       fb,
		picker:       &fakePicker{},
		pricePerTask: 5,
	}
	taskScheduled, err := w.MaybeSchedule()
	if !taskScheduled {
		t.Error("tasks were not scheduled")
	}
	if err != nil {
		t.Errorf("MaybeSchedule: %v", err)
	}
}

func TestMaybeScheduleGetBalanceFails(t *testing.T) {
	fb := &fakeBroker{balance: 50, balanceError: fmt.Errorf("not connected")}
	w := Worker{
		broker:       fb,
		picker:       &fakePicker{},
		pricePerTask: 5,
	}
	taskScheduled, err := w.MaybeSchedule()
	if err == nil {
		t.Error("Want error, got nil")
	}
	if taskScheduled {
		t.Error("tasks were scheduled but should not have")
	}

}
