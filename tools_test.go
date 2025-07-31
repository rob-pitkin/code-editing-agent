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

func TestEditFile(t *testing.T) {
	// create a test file for editing
	os.WriteFile("test_edit.txt", []byte("test1\ntest2\ntest3\ntest4\ntest5"), 0644)
	defer os.Remove("test_edit.txt")

	// happy path
	editFileInput := json.RawMessage(`{
		"path": "test_edit.txt",
		"old_str": "test1",
		"new_str": "test10"
	}`)
	result, err := EditFile(editFileInput)
	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	if result != "OK" {
		t.Fatalf("expected OK, got %s", result)
	}
	fileContent, err := os.ReadFile("test_edit.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(fileContent) != "test10\ntest2\ntest3\ntest4\ntest5" {
		t.Fatalf("expected test10, got %s", string(fileContent))
	}

	// test file doesn't exist
	editFileInput = json.RawMessage(`{
		"path": "test_edit2.txt",
		"old_str": "test1",
		"new_str": "test10"
	}`)
	_, err = EditFile(editFileInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// test file doesn't exist and old_str is empty --> create file
	editFileInput = json.RawMessage(`{
		"path": "test_edit2.txt",
		"old_str": "",
		"new_str": "test123"
	}`)
	_, err = EditFile(editFileInput)
	if err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	fileContent, err = os.ReadFile("test_edit2.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(fileContent) != "test123" {
		t.Fatalf("expected test123, got %s", string(fileContent))
	}
	defer os.Remove("test_edit2.txt")

	// test old_str doesn't match exactly once
	editFileInput = json.RawMessage(`{
		"path": "test_edit.txt",
		"old_str": "test",
		"new_str": "test99"
	}`)
	_, err = EditFile(editFileInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// test old_str and new_str are the same
	editFileInput = json.RawMessage(`{
		"path": "test_edit.txt",
		"old_str": "test1",
		"new_str": "test1"
	}`)
	_, err = EditFile(editFileInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// test old_str is empty, but file exists
	editFileInput = json.RawMessage(`{
		"path": "test_edit.txt",
		"old_str": "",
		"new_str": "test1"
	}`)
	_, err = EditFile(editFileInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// test empty file path
	editFileInput = json.RawMessage(`{
		"path": "",
		"old_str": "test1",
		"new_str": "test10"
	}`)
	_, err = EditFile(editFileInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestDeleteLines(t *testing.T) {
	// create a test file for deleting lines
	os.WriteFile("test_delete.txt", []byte("test1\ntest2\ntest3\ntest4\ntest5"), 0644)
	defer os.Remove("test_delete.txt")

	// happy path
	deleteLinesInput := json.RawMessage(`{
		"path": "test_delete.txt",
		"start_line": 1,
		"end_line": 3
	}`)
	result, err := DeleteLines(deleteLinesInput)
	if err != nil {
		t.Fatalf("failed to delete lines: %v", err)
	}
	if result != "OK" {
		t.Fatalf("expected OK, got %s", result)
	}
	fileContent, err := os.ReadFile("test_delete.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(fileContent) != "test4\ntest5" {
		t.Fatalf("expected test4\ntest5, got %s", string(fileContent))
	}

	// test out of bounds
	deleteLinesInput = json.RawMessage(`{
		"path": "test_delete.txt",
		"start_line": 1,
		"end_line": 6
	}`)
	_, err = DeleteLines(deleteLinesInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// test invalid line numbers
	deleteLinesInput = json.RawMessage(`{
		"path": "test_delete.txt",
		"start_line": 0,
		"end_line": 1
	}`)
	_, err = DeleteLines(deleteLinesInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// test invalid file path
	deleteLinesInput = json.RawMessage(`{
		"path": "",
		"start_line": 1,
		"end_line": 2
	}`)
	_, err = DeleteLines(deleteLinesInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// test start line is greater than end line
	deleteLinesInput = json.RawMessage(`{
		"path": "test_delete.txt",
		"start_line": 3,
		"end_line": 2
	}`)
	_, err = DeleteLines(deleteLinesInput)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
