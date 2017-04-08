// +build go1.8

package gorm_test

import (
	"context"
	"testing"
	"time"
)

func TestContext(t *testing.T) {
	user1 := User{Name: "RowsUser1", Age: 1, Birthday: parseTime("2000-1-1")}
	expiredCtx, cancel := context.WithDeadline(context.Background(), time.Date(2000, 1, 1, 1, 0, 0, 0, time.UTC))
	err := DB.WithContext(expiredCtx).Save(&user1).Error
	cancel()
	if err.Error() != context.DeadlineExceeded.Error() {
		t.Fatal("unexpected err:", err)
	}
}
