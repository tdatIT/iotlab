package paging

import (
	"strconv"
	"testing"
)

func TestSetSize(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		err      error
		expected int
	}{
		{
			name:     "Set Negative Size",
			value:    "-1",
			err:      strconv.ErrSyntax,
			expected: 0,
		},
		{
			name:     "Set Positive Size",
			value:    "100",
			err:      nil,
			expected: 100,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q := &ListQuery{}
			err := q.SetSize(test.value)
			if (err != nil) != (test.err != nil) {
				t.Errorf("expected error %v, got %v", test.err, err)
			}

			if q.Size != test.expected {
				t.Errorf("expected size %d, got %d", test.expected, q.Size)
			}
		})
	}
}

func TestSetPage(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		err      error
		expected int
	}{
		{
			name:     "Set Negative Page",
			value:    "-1",
			err:      strconv.ErrSyntax,
			expected: 0,
		},
		{
			name:     "Set Positive Page",
			value:    "10",
			err:      nil,
			expected: 10,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q := &ListQuery{}
			err := q.SetPage(test.value)
			if (err != nil) != (test.err != nil) {
				t.Errorf("expected error %v, got %v", test.err, err)
			}

			if q.Page != test.expected {
				t.Errorf("expected page %d, got %d", test.expected, q.Page)
			}
		})
	}
}

func TestGetOffset(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		size     int
		expected int
	}{
		{
			name:     "Get Offset with Page 1 and Size 10",
			page:     1,
			size:     10,
			expected: 0,
		},
		{
			name:     "Get Offset with Page 2 and Size 10",
			page:     2,
			size:     10,
			expected: 10,
		},
		{
			name:     "zero page and zero size uses defaults (page 1, size 15)",
			page:     0,
			size:     0,
			expected: 0,
		},
		{
			name:     "zero size uses defaultSize for offset calculation",
			page:     2,
			size:     0,
			expected: defaultSize, // (2-1) * 15
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q := &ListQuery{Page: test.page, Size: test.size}
			offset := q.GetOffset()
			if offset != test.expected {
				t.Errorf("expected offset %d, got %d", test.expected, offset)
			}
		})
	}
}

func TestGetPage(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		expected int
	}{
		{
			name:     "Get Page with Page 1",
			page:     1,
			expected: 1,
		},
		{
			name:     "Get Page with Default Page",
			page:     0,
			expected: defaultPage,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q := &ListQuery{
				Page: test.page,
			}
			page := q.GetPage()
			if page != test.expected {
				t.Errorf("expected page %d, got %d", test.expected, page)
			}
		})
	}
}

func TestGetSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		expected int
	}{
		{
			name:     "Get Size with Size 10",
			size:     10,
			expected: 10,
		},
		{
			name:     "Get Size with Default Size",
			size:     0,
			expected: defaultSize,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q := &ListQuery{
				Size: test.size,
			}
			size := q.GetSize()
			if size != test.expected {
				t.Errorf("expected size %d, got %d", test.expected, size)
			}
		})
	}
}

func TestGetTotalPages(t *testing.T) {
	tests := []struct {
		name       string
		totalCount int
		size       int
		expected   int
	}{
		{
			name:       "Get Total Pages with Total Count 100 and Size 10",
			totalCount: 100,
			size:       10,
			expected:   10,
		},
		{
			name:       "Get Total Pages with Total Count 105 and Size 10",
			totalCount: 105,
			size:       10,
			expected:   11,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q := &ListQuery{
				Size: test.size,
			}
			totalPages := q.GetTotalPages(test.totalCount)
			if totalPages != test.expected {
				t.Errorf("expected total pages %d, got %d", test.expected, totalPages)
			}
		})
	}
}

func TestGetHasMore(t *testing.T) {
	tests := []struct {
		name       string
		totalCount int
		page       int
		size       int
		expected   bool
	}{
		{
			name:       "Get Has More with Total Count 100 and Size 10 and Page 9",
			totalCount: 100,
			page:       9,
			size:       10,
			expected:   true,
		},
		{
			name:       "Get Has More with Total Count 100 and Size 10 and Page 10",
			totalCount: 100,
			page:       10,
			size:       10,
			expected:   false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q := &ListQuery{
				Size: test.size,
				Page: test.page,
			}
			hasMore := q.GetHasMore(test.totalCount)
			if hasMore != test.expected {
				t.Errorf("expected has more %v, got %v", test.expected, hasMore)
			}
		})
	}
}

func TestSetSize_emptyValue_appliesTheDefault(t *testing.T) {
	q := &ListQuery{}

	if err := q.SetSize(""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Size != defaultSize {
		t.Errorf("expected size %d, got %d", defaultSize, q.Size)
	}
}

func TestSetSize_aboveTheMaximum_isClamped(t *testing.T) {
	q := &ListQuery{}

	if err := q.SetSize("3000"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Size != maxSize {
		t.Errorf("expected size clamped to %d, got %d", maxSize, q.Size)
	}
}

func TestSetPage_emptyValue_appliesTheDefault(t *testing.T) {
	q := &ListQuery{}

	if err := q.SetPage(""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Page != defaultPage {
		t.Errorf("expected page %d, got %d", defaultPage, q.Page)
	}
}
