package copier

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type S1 struct {
	Id        string
	CreatedAt time.Time
}

type S1WithPointer struct {
	Id        *string
	CreatedAt *time.Time
}

type S2 struct {
	Id        int
	CreatedAt string
	Id2       int
}

type S3 struct {
	Embedded S1
}

type S3WithPointer struct {
	Embedded *S1
}

type S4 struct {
	Embedded S2
	S2       S2
}

func TestStruct2Struct(t *testing.T) {
	s1 := S1{
		Id:        "id1",
		CreatedAt: time.Now(),
	}
	s2 := S2{}
	err := New(IgnoreTypeError).RegisterConverter(TimeStringConverter).From(s1).To(&s2)
	assert.NoError(t, err)
	assert.Zero(t, s2.Id)
	assert.Equal(t, s1.CreatedAt.Format(time.RFC3339), s2.CreatedAt)
}

func TestEmbeddedStruct(t *testing.T) {
	s3 := S3{
		Embedded: S1{
			Id:        "test",
			CreatedAt: time.Now(),
		},
	}
	s4 := S4{}
	err := New(IgnoreTypeError).RegisterConverter(TimeStringConverter).From(s3).To(&s4)
	assert.NoError(t, err)
	assert.Equal(t, s3.Embedded.CreatedAt.Format(time.RFC3339), s4.Embedded.CreatedAt)
}

func TestEmbeddedDiffPair(t *testing.T) {
	s3 := S3{
		Embedded: S1{
			Id:        "test",
			CreatedAt: time.Now(),
		},
	}
	s4 := S4{}
	err := New(IgnoreTypeError).
		RegisterDiffPairs([]DiffPair{
			{
				Origin: "Embedded",
				Target: []string{"Embedded", "S2"},
			},
		}).
		RegisterConverter(TimeStringConverter).
		From(s3).
		To(&s4)
	assert.NoError(t, err)
	assert.Equal(t, s3.Embedded.CreatedAt.Format(time.RFC3339), s4.Embedded.CreatedAt)
	assert.Equal(t, s3.Embedded.CreatedAt.Format(time.RFC3339), s4.S2.CreatedAt)
}

func TestTransformer(t *testing.T) {
	s1 := S1{
		Id:        "123",
		CreatedAt: time.Date(2023, time.February, 1, 0, 0, 0, 0, time.Local),
	}
	s2 := S2{}
	err := New(IgnoreTypeError).RegisterTransformer("Id", func(id string) int {
		n, _ := strconv.ParseInt(id, 10, 64)
		return int(n)
	}).RegisterTransformer("CreatedAt", func(createdAt time.Time) string {
		return createdAt.Format("2006")
	}).From(s1).To(&s2)
	assert.NoError(t, err)
	assert.Equal(t, 123, s2.Id)
	assert.Equal(t, "2023", s2.CreatedAt)
}

func TestTransformerAndDiffPair(t *testing.T) {
	s1 := S1{
		Id:        "1",
		CreatedAt: time.Now(),
	}
	s2 := S2{}
	err := New(IgnoreTypeError).RegisterConverter(TimeStringConverter).RegisterDiffPairs([]DiffPair{
		{
			Origin: "Id",
			Target: []string{"Id2"},
		},
	}).RegisterTransformer("Id2", func(id string) int {
		n, _ := strconv.ParseInt(id, 10, 64)
		return int(n)
	}).From(s1).To(&s2)
	assert.NoError(t, err)
	assert.Equal(t, 1, s2.Id2)
	assert.Equal(t, 0, s2.Id)
}

func TestTimeSlice2StringSlice(t *testing.T) {
	copier := New()
	copier.RegisterConverter(TimeStringConverter)
	t1 := time.Now()
	t2 := time.Now().AddDate(1, 2, 3)
	timeSlice := []time.Time{t1, t2}
	strSlice := make([]string, 0)
	err := copier.From(timeSlice).To(&strSlice)
	assert.NoError(t, err)
	assert.Len(t, strSlice, 2)
	assert.Equal(t, t1.Format(time.RFC3339), strSlice[0])
	assert.Equal(t, t2.Format(time.RFC3339), strSlice[1])
}

func TestStringSlice2TimeSlice(t *testing.T) {
	copier := New()
	copier.RegisterConverter(StringTimeConverter)
	t1 := time.Now()
	t2 := time.Now().AddDate(1, 2, 3)
	timeSlice := make([]*time.Time, 0)
	strSlice := []string{
		t1.Format(time.RFC3339),
		t2.Format(time.RFC3339),
	}
	err := copier.From(strSlice).To(&timeSlice)
	assert.NoError(t, err)
	assert.Len(t, timeSlice, 2)
	assert.Equal(t, t1.Unix(), timeSlice[0].Unix())
	assert.Equal(t, t2.Unix(), timeSlice[1].Unix())
}

func TestStructSlice(t *testing.T) {
	slice1 := []S1{
		{
			Id:        "1",
			CreatedAt: time.Now(),
		},
		{
			Id:        "2",
			CreatedAt: time.Now().AddDate(1, 0, 0),
		},
	}
	var slice2 []S2
	err := New(IgnoreTypeError).RegisterConverter(TimeStringConverter).RegisterDiffPairs([]DiffPair{
		{
			Origin: "Id",
			Target: []string{"Id2"},
		},
	}).RegisterTransformer("Id2", func(id string) int {
		n, _ := strconv.ParseInt(id, 10, 64)
		return int(n)
	}).From(slice1).To(&slice2)
	assert.NoError(t, err)
	assert.Equal(t, len(slice1), len(slice2))
	assert.Equal(t, 1, slice2[0].Id2)
	assert.Equal(t, 2, slice2[1].Id2)
}

func TestStructPtrSlice(t *testing.T) {
	slice1 := []S1{
		{
			Id:        "1",
			CreatedAt: time.Now(),
		},
		{
			Id:        "2",
			CreatedAt: time.Now().AddDate(1, 0, 0),
		},
	}
	var slice2 []*S2
	err := New(IgnoreTypeError).RegisterConverter(TimeStringConverter).RegisterDiffPairs([]DiffPair{
		{
			Origin: "Id",
			Target: []string{"Id2"},
		},
	}).RegisterTransformer("Id2", func(id string) int {
		n, _ := strconv.ParseInt(id, 10, 64)
		return int(n)
	}).From(slice1).To(&slice2)
	assert.NoError(t, err)
	assert.Equal(t, len(slice1), len(slice2))
	assert.Equal(t, 1, slice2[0].Id2)
	assert.Equal(t, 2, slice2[1].Id2)
}

type S5 struct {
	S1 S1
}

type S6 struct {
	S5 S5
}

func TestMultiField(t *testing.T) {
	s5 := S5{
		S1: S1{
			Id:        "123",
			CreatedAt: time.Now(),
		},
	}
	s6 := S6{}
	err := New(IgnoreTypeError).RegisterDiffPairs([]DiffPair{
		{
			Origin: "S1.Id",
			Target: []string{"S5.S1.Id"},
		},
	}).From(s5).To(&s6)
	assert.NoError(t, err)
	assert.Equal(t, s5.S1.Id, s6.S5.S1.Id)
}

func TestMultiFieldWithTransformer(t *testing.T) {
	s5 := S5{
		S1: S1{
			Id:        "123",
			CreatedAt: time.Now(),
		},
	}
	s6 := S6{}
	err := New(IgnoreTypeError).RegisterDiffPairs([]DiffPair{
		{
			Origin: "S1.Id",
			Target: []string{"S5.S1.Id"},
		},
	}).RegisterTransformer("S5.S1.Id", func(id string) string {
		return fmt.Sprintf("test_%s", id)
	}).From(s5).To(&s6)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("test_%s", s5.S1.Id), s6.S5.S1.Id)
}

func TestPartialCopy(t *testing.T) {
	s5 := S5{
		S1: S1{
			Id:        "123",
			CreatedAt: time.Now(),
		},
	}
	s6 := S6{}
	err := New(IgnoreTypeError).From(s5.S1).To(&(s6.S5.S1))
	assert.NoError(t, err)
	assert.Equal(t, s5.S1.Id, s6.S5.S1.Id)
	assert.Equal(t, s5.S1.CreatedAt, s6.S5.S1.CreatedAt)
}

func TestCopyWithPointer(t *testing.T) {
	from := S1{
		Id:        "123",
		CreatedAt: time.Now(),
	}
	to := S1WithPointer{}
	fn := func() {
		err := New(IgnoreTypeError).From(from).To(&to)
		assert.NoError(t, err)
		assert.NotNil(t, to.Id)
		assert.Equal(t, from.Id, *to.Id)
		assert.NotNil(t, to.CreatedAt)
		assert.Equal(t, from.CreatedAt, *to.CreatedAt)
	}
	assert.NotPanics(t, fn)
}

func TestCopyWithStructPointer(t *testing.T) {
	from := S3{
		Embedded: S1{
			Id:        "123",
			CreatedAt: time.Now(),
		},
	}
	to := S3WithPointer{}
	fn := func() {
		err := New(IgnoreTypeError).From(from).To(&to)
		assert.NoError(t, err)
		assert.NotNil(t, to.Embedded)
		assert.Equal(t, from.Embedded.Id, to.Embedded.Id)
		assert.Equal(t, from.Embedded.CreatedAt, to.Embedded.CreatedAt)
	}
	assert.NotPanics(t, fn)
}

func TestCopyEmptyStrToTime(t *testing.T) {
	from := S2{}
	to := S1{}
	fn := func() {
		err := New(
			IgnoreTypeError,
			IgnoreZeroValue,
		).RegisterConverter(StringTimeConverter).From(from).To(&to)
		assert.NoError(t, err)
		assert.True(t, to.CreatedAt.IsZero())
	}
	assert.NotPanics(t, fn)
}
