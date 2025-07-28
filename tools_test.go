package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestReadLines(t *testing.T) {
	// test reading 1 line
	readLinesInput := json.RawMessage(`{
		"path": "test.txt",
		"start_line": 1,
		"end_line": 2
	}`)
	result, err := ReadLines(readLinesInput)
	if err != nil {
		t.Fatalf("failed to read lines: %v", err)
	}

	var lines []string
	err = json.Unmarshal([]byte(result), &lines)
	if err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	expected := []string{"<line-1>test1</line-1>"}

	if !reflect.DeepEqual(lines, expected) {
		t.Fatalf("expected %v, got %v", expected, lines)
	}

	// test reading 0 lines
	readLinesInput = json.RawMessage(`{
		"path": "test.txt",
		"start_line": 2,
		"end_line": 2
	}`)
	result, err = ReadLines(readLinesInput)
	if err != nil {
		t.Fatalf("failed to read lines: %v", err)
	}

	err = json.Unmarshal([]byte(result), &lines)
	if err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	expected = []string{}
	if !reflect.DeepEqual(lines, expected) {
		t.Fatalf("expected %v, got %v", expected, lines)
	}

	// test out of bounds
	readLinesInput = json.RawMessage(`{
		"path": "test.txt",
		"start_line": 3,
		"end_line": 2
	}`)
	_, err = ReadLines(readLinesInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// test reading more lines than the file has
	readLinesInput = json.RawMessage(`{
		"path": "test.txt",
		"start_line": 1,
		"end_line": 5
	}`)
	_, err = ReadLines(readLinesInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// test reading the whole file
	readLinesInput = json.RawMessage(`{
		"path": "test.txt",
		"start_line": 1,
		"end_line": 4
	}`)
	result, err = ReadLines(readLinesInput)
	if err != nil {
		t.Fatalf("failed to read lines: %v", err)
	}

	err = json.Unmarshal([]byte(result), &lines)
	if err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	expected = []string{"<line-1>test1</line-1>", "<line-2>test2</line-2>", "<line-3>test3</line-3>"}
	if !reflect.DeepEqual(lines, expected) {
		t.Fatalf("expected %v, got %v", expected, lines)
	}

	// test file doesn't exist
	readLinesInput = json.RawMessage(`{
		"path":      "test2.txt",
		"start_line": 1,
		"end_line":   2
	}`)
	_, err = ReadLines(readLinesInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func resetTestFile() {
	os.WriteFile("test_edit.txt", []byte("test line one\ntest line two\ntest line three\ntest line four\ntest line five"), 0644)
}

func TestEditFileEdgeCases(t *testing.T) {
	// Test what happens when edits is missing
	missingEdits := json.RawMessage(`{"path": "test.txt"}`)
	_, err := EditFile(missingEdits)
	if err == nil {
		t.Fatal("Expected error for missing edits field")
	}
	t.Logf("Missing edits error: %v", err)

	// Test what happens when edits is empty
	emptyEdits := json.RawMessage(`{"path": "test.txt", "edits": []}`)
	_, err = EditFile(emptyEdits)
	if err == nil {
		t.Fatal("Expected error for empty edits array")
	}
	t.Logf("Empty edits error: %v", err)

	// Test malformed JSON
	badJson := json.RawMessage(`{"path": "test.txt", "edits": `)
	_, err = EditFile(badJson)
	if err == nil {
		t.Fatal("Expected error for malformed JSON")
	}
	t.Logf("Malformed JSON error: %v", err)

	// Test edit with missing required fields
	os.WriteFile("test_temp.txt", []byte("line 1\nline 2\n"), 0644)
	defer os.Remove("test_temp.txt")

	missingLineNumber := json.RawMessage(`{
		"path": "test_temp.txt",
		"edits": [{"operation_type": "replace_line", "new_content": ["new line"]}]
	}`)
	_, err = EditFile(missingLineNumber)
	if err == nil {
		t.Fatal("Expected error for missing line_number")
	}
	t.Logf("Missing line_number error: %v", err)
}

func TestEditFile(t *testing.T) {
	defer func() {
		os.Remove("test_edit.txt")
	}()

	// init file with 5 lines
	os.WriteFile("test_edit.txt", []byte("test line one\ntest line two\ntest line three\ntest line four\ntest line five"), 0644)

	editFileReplaceLineInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "replace_line",
				"line_number": 1,
				"new_content": ["test line two"]
			}
		]
	}`)

	expectedReplaceLineResult := "test line two\ntest line two\ntest line three\ntest line four\ntest line five"

	_, err := EditFile(editFileReplaceLineInput)
	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	// Read file directly for comparison
	content, err := os.ReadFile("test_edit.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	readFileResult := string(content)
	if readFileResult != expectedReplaceLineResult {
		t.Fatalf("expected %v, got %v", expectedReplaceLineResult, readFileResult)
	}

	resetTestFile()

	editFileInsertLineBeforeInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "insert_line_before",
				"line_number": 1,
				"new_content": ["test line zero"]
			}
		]
	}`)

	expectedInsertLineBeforeResult := "test line zero\ntest line one\ntest line two\ntest line three\ntest line four\ntest line five"

	_, err = EditFile(editFileInsertLineBeforeInput)

	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	content, err = os.ReadFile("test_edit.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	readFileResult = string(content)
	if readFileResult != expectedInsertLineBeforeResult {
		t.Fatalf("expected %v, got %v", expectedInsertLineBeforeResult, readFileResult)
	}

	resetTestFile()

	editFileInsertLineAfterInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "insert_line_after",
				"line_number": 1,
				"new_content": ["test line zero"]
			}
		]
	}`)

	expectedInsertLineAfterResult := "test line one\ntest line zero\ntest line two\ntest line three\ntest line four\ntest line five"

	_, err = EditFile(editFileInsertLineAfterInput)

	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	content, err = os.ReadFile("test_edit.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	readFileResult = string(content)
	if readFileResult != expectedInsertLineAfterResult {
		t.Fatalf("expected %v, got %v", expectedInsertLineAfterResult, readFileResult)
	}

	resetTestFile()

	editFileDeleteLineInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "delete_line",
				"line_number": 1
			}
		]
	}`)

	expectedDeleteLineResult := "test line two\ntest line three\ntest line four\ntest line five"

	_, err = EditFile(editFileDeleteLineInput)
	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	content, err = os.ReadFile("test_edit.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	readFileResult = string(content)
	if readFileResult != expectedDeleteLineResult {
		t.Fatalf("expected %v, got %v", expectedDeleteLineResult, readFileResult)
	}

	resetTestFile()

	editFileReplaceStringInLineInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "replace_string_in_line",
				"line_number": 1,
				"old_string": "one",
				"new_string": "zero",
				"count": -1
			}
		]
	}`)

	expectedReplaceStringInLineResult := "test line zero\ntest line two\ntest line three\ntest line four\ntest line five"

	_, err = EditFile(editFileReplaceStringInLineInput)
	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	content, err = os.ReadFile("test_edit.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	readFileResult = string(content)
	if readFileResult != expectedReplaceStringInLineResult {
		t.Fatalf("expected %v, got %v", expectedReplaceStringInLineResult, readFileResult)
	}

	resetTestFile()

	editFileAppendToFileInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "append_to_file",
				"line_number": 1,
				"new_content": ["test line six"]
			}
		]
	}`)

	expectedAppendToFileResult := "test line one\ntest line two\ntest line three\ntest line four\ntest line five\ntest line six"

	_, err = EditFile(editFileAppendToFileInput)
	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	content, err = os.ReadFile("test_edit.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	readFileResult = string(content)
	if readFileResult != expectedAppendToFileResult {
		t.Fatalf("expected %v, got %v", expectedAppendToFileResult, readFileResult)
	}

	resetTestFile()

	editFileMultipleEditsInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "replace_line",
				"line_number": 1,
				"new_content": ["test line zero"]
			},
			{
				"operation_type": "insert_line_before",
				"line_number": 1,
				"new_content": ["test line one"]
			},
			{
				"operation_type": "insert_line_after",
				"line_number": 5,
				"new_content": ["test line four point five"]
			},
			{
				"operation_type": "delete_line",
				"line_number": 7
			},
			{
				"operation_type": "replace_string_in_line",
				"line_number": 5,
				"old_string": "test line four",
				"new_string": "test line three point five",
				"count": -1
			},
			{
				"operation_type": "append_to_file",
				"new_content": ["test line seven"]
			}
		]
	}`)

	expectedMultipleEditsResult := "test line one\ntest line zero\ntest line two\ntest line three\ntest line three point five\ntest line four point five\ntest line seven"

	_, err = EditFile(editFileMultipleEditsInput)
	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	content, err = os.ReadFile("test_edit.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	readFileResult = string(content)
	if readFileResult != expectedMultipleEditsResult {
		t.Fatalf("expected %v, got %v", expectedMultipleEditsResult, readFileResult)
	}

	editFileInvalidInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "replace_line"
			}
		]
	}`)
	_, err = EditFile(editFileInvalidInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	editFileInvalidOperationTypeInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "invalid_operation_type"
			}
		]
	}`)
	_, err = EditFile(editFileInvalidOperationTypeInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	editFileInvalidLineNumberInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": [
			{
				"operation_type": "replace_line",
				"line_number": 0,
				"new_content": ["test line zero"]
			}
		]
	}`)
	_, err = EditFile(editFileInvalidLineNumberInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// Test empty edits array
	editFileEmptyEditsInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"edits": []
	}`)
	_, err = EditFile(editFileEmptyEditsInput)
	if err == nil {
		t.Fatalf("expected error for empty edits array, got nil")
	}

	// Test missing edits field
	editFileMissingEditsInput := json.RawMessage(`{
		"path": "test_edit.txt"
	}`)
	_, err = EditFile(editFileMissingEditsInput)
	if err == nil {
		t.Fatalf("expected error for missing edits field, got nil")
	}
}

func TestReplaceLineMultipleLines(t *testing.T) {
	// Create a test file
	os.WriteFile("test_multiline.txt", []byte("line 1\nline 2\nline 3\n"), 0644)
	defer os.Remove("test_multiline.txt")

	// Test replace_line with multiple lines in new_content
	editFileReplaceMultipleLines := json.RawMessage(`{
		"path": "test_multiline.txt",
		"edits": [
			{
				"operation_type": "replace_line",
				"line_number": 2,
				"new_content": ["replaced line 1", "replaced line 2", "replaced line 3"]
			}
		]
	}`)

	expectedResult := "line 1\nreplaced line 1\nreplaced line 2\nreplaced line 3\nline 3\n"

	_, err := EditFile(editFileReplaceMultipleLines)
	if err != nil {
		t.Fatalf("failed to edit file with multiple line replacement: %v", err)
	}

	// Read file directly for comparison
	content, err := os.ReadFile("test_multiline.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	readFileResult := string(content)
	if readFileResult != expectedResult {
		t.Fatalf("expected %q, got %q", expectedResult, readFileResult)
	}
}
