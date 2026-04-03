package core

import (
	"context"
	"fmt"
)

func runEntryStateAction(ctx context.Context, id int64, format string, action func(context.Context, int64) error) error {
	if err := action(ctx, id); err != nil {
		return fmt.Errorf(format, id, err)
	}
	return nil
}

func runEntryStateBatchAction(ctx context.Context, ids []int64, format string, action func(context.Context, []int64) error) error {
	if len(ids) == 0 {
		return nil
	}
	if err := action(ctx, ids); err != nil {
		return fmt.Errorf(format, err)
	}
	return nil
}
