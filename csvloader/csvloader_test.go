package csvloader

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"gorm.io/datatypes"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestOpenCSV(t *testing.T) {
	// 正常系テスト: 有効なCSVファイルを読み込むutf-8
	t.Run("valid CSV", func(t *testing.T) {
		csvData := `header1,header2,header3
value1,value2,value3
value4,value5,value6`
		filePath := "valid.csv"
		err := os.WriteFile(filePath, []byte(csvData), 0644)
		require.NoError(t, err) // ファイル書き込みのエラーハンドリング
		defer os.Remove(filePath)

		df, err := OpenCSV(filePath, "utf-8")
		assert.NoError(t, err)
		assert.NotNil(t, df)
		assert.Equal(t, 3, len(df.Headers))
		assert.Equal(t, 2, len(df.Records))
		assert.Equal(t, "value3", df.Records[0][2])
	})

	// 正常系テスト: 有効なCSVファイルを読み込むshift-jis
	t.Run("valid CSV", func(t *testing.T) {
		csvData := `header1,header2,header3
value1,value2,value3
value4,value5,value6`
		filePath := "valid.csv"

		var buf bytes.Buffer
		writer := transform.NewWriter(&buf, japanese.ShiftJIS.NewEncoder())
		_, err := writer.Write([]byte(csvData))
		require.NoError(t, err)
		err = os.WriteFile(filePath, buf.Bytes(), 0644)
		require.NoError(t, err) // ファイル書き込みのエラーハンドリング
		defer os.Remove(filePath)

		df, err := OpenCSV(filePath, "shift-jis")
		assert.NoError(t, err)
		assert.NotNil(t, df)
		assert.Equal(t, 3, len(df.Headers))
		assert.Equal(t, 2, len(df.Records))
		assert.Equal(t, "value3", df.Records[0][2])
	})

	// 異常系テスト: ファイルが存在しない場合
	t.Run("file not found", func(t *testing.T) {
		_, err := OpenCSV("not_exist.csv", "utf-8")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open file")
	})

	// 異常系テスト: 定義されていないencoding
	t.Run("encoding not found", func(t *testing.T) {
		csvData := `header1,header2,header3
value1,value2,value3
value4,value5,value6`
		filePath := "valid.csv"
		err := os.WriteFile(filePath, []byte(csvData), 0644)
		require.NoError(t, err) // ファイル書き込みのエラーハンドリング
		defer os.Remove(filePath)

		_, err = OpenCSV(filePath, "hoge")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported encoding")
	})

	// 異常系テスト: BOM付きかつ不正なヘッダーのファイル
	t.Run("invalid header", func(t *testing.T) {
		csvData := "\ufeff"
		filePath := "invalid_header.csv"
		err := os.WriteFile(filePath, []byte(csvData), 0644)
		require.NoError(t, err)
		defer os.Remove(filePath)

		_, err = OpenCSV(filePath, "utf-8")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read headers") // 修正
	})

	// 異常系テスト: ヘッダーは正しいがデータ読み込みに失敗
	t.Run("invalid body", func(t *testing.T) {
		csvData := "\ufeffheader1,header2,header3\nvalue1,value2"
		filePath := "invalid_body.csv"
		err := os.WriteFile(filePath, []byte(csvData), 0644)
		require.NoError(t, err)
		defer os.Remove(filePath)

		_, err = OpenCSV(filePath, "utf-8")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read records") // 修正
	})

	// BOM付き正常データのテスト
	t.Run("valid CSV with BOM", func(t *testing.T) {
		csvData := "\ufeffheader1,header2,header3\nvalue1,value2,value3\nvalue4,value5,value6"
		filePath := "valid_with_bom.csv"
		err := os.WriteFile(filePath, []byte(csvData), 0644)
		require.NoError(t, err)
		defer os.Remove(filePath)

		df, err := OpenCSV(filePath, "utf-8")
		assert.NoError(t, err)
		assert.NotNil(t, df)
		assert.Equal(t, 3, len(df.Headers))
		assert.Equal(t, 2, len(df.Records))
		assert.Equal(t, "value3", df.Records[0][2])
	})
}

func TestDataFrame_ToJSON(t *testing.T) {
	type fields struct {
		Headers map[string]int
		Records [][]string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "empty DataFrame",
			fields: fields{
				Headers: map[string]int{},
				Records: [][]string{},
			},
			want:    "[]",
			wantErr: assert.NoError,
		},
		{
			name: "single row",
			fields: fields{
				Headers: map[string]int{"name": 0, "age": 1},
				Records: [][]string{{"Alice", "30"}},
			},
			want: `[
  {
    "name": "Alice",
    "age": "30"
  }
]`,
			wantErr: assert.NoError,
		},
		{
			name: "multiple rows",
			fields: fields{
				Headers: map[string]int{"name": 0, "age": 1},
				Records: [][]string{
					{"Alice", "30"},
					{"Bob", "25"},
				},
			},
			want: `[
  {
    "name": "Alice",
    "age": "30"
  },
  {
    "name": "Bob",
    "age": "25"
  }
]`,
			wantErr: assert.NoError,
		},
		{
			name: "row with missing column data",
			fields: fields{
				Headers: map[string]int{"name": 0, "age": 1, "city": 2},
				Records: [][]string{
					{"Alice", "30"}, // "city" のデータが不足
				},
			},
			want: `[
  {
    "name": "Alice",
    "age": "30",
    "city": ""
  }
]`,
			wantErr: assert.NoError,
		},
		{
			name: "row with extra column data",
			fields: fields{
				Headers: map[string]int{"name": 0, "age": 1},
				Records: [][]string{
					{"Alice", "30", "New York"}, // "New York" は無視される
				},
			},
			want: `[
  {
    "name": "Alice",
    "age": "30"
  }
]`,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := &DataFrame{
				Headers: tt.fields.Headers,
				Records: tt.fields.Records,
			}
			got, err := df.ToJSON()
			if !tt.wantErr(t, err, fmt.Sprintf("ToJSON()")) {
				return
			}
			assert.JSONEq(t, tt.want, got, "ToJSON() output mismatch")
		})
	}
}

func TestDataFrame_GetString(t *testing.T) {
	type fields struct {
		headers map[string]int
		records [][]string
	}
	type args struct {
		rowIndex   int
		columnName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Valid string",
			fields: fields{
				headers: map[string]int{"name": 0},
				records: [][]string{{"John"}, {"Doe"}},
			},
			args:    args{rowIndex: 0, columnName: "name"},
			want:    "John",
			wantErr: false,
		},
		{
			name: "Invalid column name",
			fields: fields{
				headers: map[string]int{"name": 0},
				records: [][]string{{"John"}, {"Doe"}},
			},
			args:    args{rowIndex: 0, columnName: "age"},
			want:    "",
			wantErr: true,
		},
		{
			name: "Row index out of bounds",
			fields: fields{
				headers: map[string]int{"name": 0},
				records: [][]string{{"John"}, {"Doe"}},
			},
			args:    args{rowIndex: 2, columnName: "name"},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := &DataFrame{
				Headers: tt.fields.headers,
				Records: tt.fields.records,
			}
			got, err := df.GetString(tt.args.rowIndex, tt.args.columnName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetString() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataFrame_GetStringPtr(t *testing.T) {
	type fields struct {
		headers map[string]int
		records [][]string
	}
	type args struct {
		rowIndex   int
		columnName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *string
		wantErr bool
	}{
		{
			name: "Valid string",
			fields: fields{
				headers: map[string]int{"name": 0},
				records: [][]string{{"John"}, {"Doe"}},
			},
			args:    args{rowIndex: 0, columnName: "name"},
			want:    stringPtr("John"),
			wantErr: false,
		},
		{
			name: "Empty string",
			fields: fields{
				headers: map[string]int{"name": 0},
				records: [][]string{{""}},
			},
			args:    args{rowIndex: 0, columnName: "name"},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Column does not exist",
			fields: fields{
				headers: map[string]int{"name": 0},
				records: [][]string{{"John"}},
			},
			args:    args{rowIndex: 0, columnName: "age"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := &DataFrame{
				Headers: tt.fields.headers,
				Records: tt.fields.records,
			}
			got, err := df.GetStringPtr(tt.args.rowIndex, tt.args.columnName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStringPtr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetStringPtr() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// Utility functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func TestDataFrame_GetInt(t *testing.T) {
	type fields struct {
		headers map[string]int
		records [][]string
	}
	type args struct {
		rowIndex   int
		columnName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "Valid integer",
			fields: fields{
				headers: map[string]int{"age": 0},
				records: [][]string{{"25"}},
			},
			args:    args{rowIndex: 0, columnName: "age"},
			want:    25,
			wantErr: false,
		},
		{
			name: "Invalid integer format",
			fields: fields{
				headers: map[string]int{"age": 0},
				records: [][]string{{"invalid"}},
			},
			args:    args{rowIndex: 0, columnName: "age"},
			want:    0,
			wantErr: true,
		},
		{
			name: "Column does not exist",
			fields: fields{
				headers: map[string]int{"age": 0},
				records: [][]string{{"25"}},
			},
			args:    args{rowIndex: 0, columnName: "height"},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := &DataFrame{
				Headers: tt.fields.headers,
				Records: tt.fields.records,
			}
			got, err := df.GetInt(tt.args.rowIndex, tt.args.columnName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetInt() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataFrame_GetIntPtr(t *testing.T) {
	type fields struct {
		headers map[string]int
		records [][]string
	}
	type args struct {
		rowIndex   int
		columnName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *int
		wantErr bool
	}{
		{
			name: "Valid integer",
			fields: fields{
				headers: map[string]int{"age": 0},
				records: [][]string{{"25"}},
			},
			args:    args{rowIndex: 0, columnName: "age"},
			want:    intPtr(25),
			wantErr: false,
		},
		{
			name: "Empty value",
			fields: fields{
				headers: map[string]int{"age": 0},
				records: [][]string{{""}},
			},
			args:    args{rowIndex: 0, columnName: "age"},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Invalid integer format",
			fields: fields{
				headers: map[string]int{"age": 0},
				records: [][]string{{"invalid"}},
			},
			args:    args{rowIndex: 0, columnName: "age"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Column does not exist",
			fields: fields{
				headers: map[string]int{"age": 0},
				records: [][]string{{"25"}},
			},
			args:    args{rowIndex: 0, columnName: "height"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := &DataFrame{
				Headers: tt.fields.headers,
				Records: tt.fields.records,
			}
			got, err := df.GetIntPtr(tt.args.rowIndex, tt.args.columnName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIntPtr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetIntPtr() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// Utility function for creating int pointers
func intPtr(i int) *int {
	return &i
}

func TestDataFrame_GetFloat(t *testing.T) {
	type fields struct {
		headers map[string]int
		records [][]string
	}
	type args struct {
		rowIndex   int
		columnName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    float64
		wantErr bool
	}{
		{
			name: "Valid float",
			fields: fields{
				headers: map[string]int{"price": 0},
				records: [][]string{{"123.45"}},
			},
			args:    args{rowIndex: 0, columnName: "price"},
			want:    123.45,
			wantErr: false,
		},
		{
			name: "Invalid float format",
			fields: fields{
				headers: map[string]int{"price": 0},
				records: [][]string{{"invalid"}},
			},
			args:    args{rowIndex: 0, columnName: "price"},
			want:    0.0,
			wantErr: true,
		},
		{
			name: "Column does not exist",
			fields: fields{
				headers: map[string]int{"price": 0},
				records: [][]string{{"123.45"}},
			},
			args:    args{rowIndex: 0, columnName: "weight"},
			want:    0.0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := &DataFrame{
				Headers: tt.fields.headers,
				Records: tt.fields.records,
			}
			got, err := df.GetFloat(tt.args.rowIndex, tt.args.columnName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFloat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetFloat() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataFrame_GetFloatPtr(t *testing.T) {
	type fields struct {
		headers map[string]int
		records [][]string
	}
	type args struct {
		rowIndex   int
		columnName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *float64
		wantErr bool
	}{
		{
			name: "Valid float",
			fields: fields{
				headers: map[string]int{"price": 0},
				records: [][]string{{"123.45"}},
			},
			args:    args{rowIndex: 0, columnName: "price"},
			want:    floatPtr(123.45),
			wantErr: false,
		},
		{
			name: "Empty value",
			fields: fields{
				headers: map[string]int{"price": 0},
				records: [][]string{{""}},
			},
			args:    args{rowIndex: 0, columnName: "price"},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Invalid float format",
			fields: fields{
				headers: map[string]int{"price": 0},
				records: [][]string{{"invalid"}},
			},
			args:    args{rowIndex: 0, columnName: "price"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Column does not exist",
			fields: fields{
				headers: map[string]int{"price": 0},
				records: [][]string{{"123.45"}},
			},
			args:    args{rowIndex: 0, columnName: "weight"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := &DataFrame{
				Headers: tt.fields.headers,
				Records: tt.fields.records,
			}
			got, err := df.GetFloatPtr(tt.args.rowIndex, tt.args.columnName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFloatPtr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetFloatPtr() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// Utility function for creating float pointers
func floatPtr(f float64) *float64 {
	return &f
}

func TestDataFrame_GetDate(t *testing.T) {
	type fields struct {
		headers map[string]int
		records [][]string
	}
	type args struct {
		rowIndex   int
		columnName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    datatypes.Date
		wantErr bool
	}{
		{
			name: "Valid date",
			fields: fields{
				headers: map[string]int{"birthdate": 0},
				records: [][]string{{"20240101"}},
			},
			args:    args{rowIndex: 0, columnName: "birthdate"},
			want:    datatypes.Date(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			wantErr: false,
		},
		{
			name: "Invalid date format",
			fields: fields{
				headers: map[string]int{"birthdate": 0},
				records: [][]string{{"invalid"}},
			},
			args:    args{rowIndex: 0, columnName: "birthdate"},
			want:    datatypes.Date{},
			wantErr: true,
		},
		{
			name: "Column does not exist",
			fields: fields{
				headers: map[string]int{"birthdate": 0},
				records: [][]string{{"20240101"}},
			},
			args:    args{rowIndex: 0, columnName: "unknown"},
			want:    datatypes.Date{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := &DataFrame{
				Headers: tt.fields.headers,
				Records: tt.fields.records,
			}
			got, err := df.GetDate(tt.args.rowIndex, tt.args.columnName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDate() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataFrame_GetDatePtr(t *testing.T) {
	// Arrange
	df := &DataFrame{
		Headers: map[string]int{"valid_date": 0, "empty_date": 1, "invalid_date": 2},
		Records: [][]string{
			{"20240101", "", "invalid"},
		},
	}

	tests := []struct {
		name       string
		rowIndex   int
		columnName string
		want       *datatypes.Date
		wantErr    bool
	}{
		{
			name:       "Valid date",
			rowIndex:   0,
			columnName: "valid_date",
			want:       datePtr(datatypes.Date(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))),
			wantErr:    false,
		},
		{
			name:       "Empty date",
			rowIndex:   0,
			columnName: "empty_date",
			want:       nil,
			wantErr:    false,
		},
		{
			name:       "Invalid date format",
			rowIndex:   0,
			columnName: "invalid_date",
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "Non-existent column",
			rowIndex:   0,
			columnName: "non_existent",
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "Row index out of range",
			rowIndex:   1,
			columnName: "valid_date",
			want:       nil,
			wantErr:    true,
		},
	}

	// Act and Assert
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := df.GetDatePtr(tt.rowIndex, tt.columnName)

			// Check for errors
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDatePtr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check for expected values
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDatePtr() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// Utility function for creating date pointers
func datePtr(d datatypes.Date) *datatypes.Date {
	return &d
}

func TestDataFrame_GetTime(t *testing.T) {
	type fields struct {
		headers map[string]int
		records [][]string
	}
	type args struct {
		rowIndex   int
		columnName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    datatypes.Time
		wantErr bool
	}{
		{
			name: "Valid time",
			fields: fields{
				headers: map[string]int{"startTime": 0},
				records: [][]string{{"14:30:00"}},
			},
			args:    args{rowIndex: 0, columnName: "startTime"},
			want:    datatypes.NewTime(14, 30, 0, 0),
			wantErr: false,
		},
		{
			name: "Invalid time format",
			fields: fields{
				headers: map[string]int{"startTime": 0},
				records: [][]string{{"invalid"}},
			},
			args:    args{rowIndex: 0, columnName: "startTime"},
			want:    datatypes.NewTime(0, 0, 0, 0),
			wantErr: true,
		},
		{
			name: "Column does not exist",
			fields: fields{
				headers: map[string]int{"startTime": 0},
				records: [][]string{{"14:30:00"}},
			},
			args:    args{rowIndex: 0, columnName: "endTime"},
			want:    datatypes.NewTime(0, 0, 0, 0),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := &DataFrame{
				Headers: tt.fields.headers,
				Records: tt.fields.records,
			}
			got, err := df.GetTime(tt.args.rowIndex, tt.args.columnName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTime() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataFrame_GetTimePtr(t *testing.T) {
	type fields struct {
		headers map[string]int
		records [][]string
	}
	type args struct {
		rowIndex   int
		columnName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *datatypes.Time
		wantErr bool
	}{
		{
			name: "Valid time",
			fields: fields{
				headers: map[string]int{"startTime": 0},
				records: [][]string{{"14:30:00"}},
			},
			args:    args{rowIndex: 0, columnName: "startTime"},
			want:    timePtr(datatypes.NewTime(14, 30, 0, 0)),
			wantErr: false,
		},
		{
			name: "Empty value",
			fields: fields{
				headers: map[string]int{"startTime": 0},
				records: [][]string{{""}},
			},
			args:    args{rowIndex: 0, columnName: "startTime"},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Invalid time format",
			fields: fields{
				headers: map[string]int{"startTime": 0},
				records: [][]string{{"invalid"}},
			},
			args:    args{rowIndex: 0, columnName: "startTime"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Column does not exist",
			fields: fields{
				headers: map[string]int{"startTime": 0},
				records: [][]string{{"14:30:00"}},
			},
			args:    args{rowIndex: 0, columnName: "endTime"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := &DataFrame{
				Headers: tt.fields.headers,
				Records: tt.fields.records,
			}
			got, err := df.GetTimePtr(tt.args.rowIndex, tt.args.columnName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTimePtr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTimePtr() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// Utility function for creating time pointers
func timePtr(t datatypes.Time) *datatypes.Time {
	return &t
}
