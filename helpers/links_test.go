package helpers

import (
	"testing"
)

func TestPageCount(t *testing.T) {

	var (
		total        int64
		limit        int64
		correctValue int64
		result       int64
	)

	message := "GetPageCount(%d, %d) = %d should be %d"

	// limit = 25 (default)
	limit = DefaultQueryLimit

	total = 0
	correctValue = 0
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 1
	correctValue = 1
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 24
	correctValue = 1
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 25
	correctValue = 1
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 26
	correctValue = 2
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 49
	correctValue = 2
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 50
	correctValue = 2
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 51
	correctValue = 3
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	// limit = 5
	limit = 5

	total = 0
	correctValue = 0
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 1
	correctValue = 1
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 4
	correctValue = 1
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 5
	correctValue = 1
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 6
	correctValue = 2
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 9
	correctValue = 2
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 10
	correctValue = 2
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 11
	correctValue = 3
	result = GetPageCount(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}
}

func TestGetMaxOffset(t *testing.T) {

	var (
		total        int64
		limit        int64
		correctValue int64
		result       int64
	)

	message := "GetMaxOffset(%d, %d) = %d should be %d"

	// limit = 25 (default)
	limit = DefaultQueryLimit

	total = 0
	correctValue = 0
	result = GetMaxOffset(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 24
	correctValue = 0
	result = GetMaxOffset(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 25
	correctValue = 0
	result = GetMaxOffset(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 26
	correctValue = 25
	result = GetMaxOffset(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 49
	correctValue = 25
	result = GetMaxOffset(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 50
	correctValue = 25
	result = GetMaxOffset(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 51
	correctValue = 50
	result = GetMaxOffset(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	// limit = 25 (default)
	limit = 5

	total = 0
	correctValue = 0
	result = GetMaxOffset(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 4
	correctValue = 0
	result = GetMaxOffset(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 5
	correctValue = 0
	result = GetMaxOffset(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 6
	correctValue = 5
	result = GetMaxOffset(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 9
	correctValue = 5
	result = GetMaxOffset(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 10
	correctValue = 5
	result = GetMaxOffset(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

	total = 11
	correctValue = 10
	result = GetMaxOffset(total, limit)
	if result != correctValue {
		t.Errorf(message, total, limit, result, correctValue)
	}

}
