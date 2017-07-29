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
	balance         int
	getBalanceError error
}

func (b *fakeBroker) GetBalance() (int, error) {
	return b.balance, b.getBalanceError
}

func (b *fakeBroker) ChangeBalance(delta int) error {
	b.balance -= delta
	return nil
}

func TestMaybeSchedule(t *testing.T) {
	var tests = []struct {
		msg           string
		w             Worker
		taskScheduled bool
		wantError     bool
	}{
		{
			msg: "balance ok",
			w: Worker{
				broker:       &fakeBroker{balance: 50},
				picker:       &fakePicker{},
				pricePerTask: 5,
			},
			taskScheduled: true,
			wantError:     false,
		}, {
			msg: "not enough balance",
			w: Worker{
				broker:       &fakeBroker{balance: 3},
				picker:       &fakePicker{},
				pricePerTask: 5,
			},
			taskScheduled: false,
			wantError:     false,
		}, {
			msg: "error checking balance",
			w: Worker{
				broker: &fakeBroker{
					balance:         50,
					getBalanceError: fmt.Errorf("not connected"),
				},
				picker:       &fakePicker{},
				pricePerTask: 5,
			},
			taskScheduled: false,
			wantError:     true,
		},
	}
	for _, test := range tests {
		taskScheduled, err := test.w.MaybeSchedule()
		if got, want := taskScheduled, test.taskScheduled; got != want {
			t.Errorf("[%s] task scheduled, got=%t, want=%t", test.msg, got, want)
		}
		if got, want := err != nil, test.wantError; got != want {
			t.Errorf("[%s]wantError, got=%t, want=%t", test.msg, got, want)
		}
	}
}
