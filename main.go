package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/invopop/jsonschema"
)

func main() {
	client := anthropic.NewClient()

	scanner := bufio.NewScanner(os.Stdin)
	getUserMessage := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	tools := []ToolDefinition{ReadFileDefinition, ListFilesDefinition, EditFileDefinition, ReadLinesDefinition}
	agent := NewAgent(&client, getUserMessage, tools)
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

func NewAgent(client *anthropic.Client, getUserMessage func() (string, bool), tools []ToolDefinition) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          tools,
	}
}

type Agent struct {
	client         *anthropic.Client
	getUserMessage func() (string, bool)
	tools          []ToolDefinition
}

func (a *Agent) Run(ctx context.Context) error {
	conversation := []anthropic.MessageParam{}

	fmt.Println("Chat with Claude (use 'ctrl-c' to quit)")

	readUserInput := true
	for {
		if readUserInput {
			fmt.Print("\u001b[94mYou\u001b[0m: ")
			userInput, ok := a.getUserMessage()
			if !ok {
				break
			}

			userMessage := anthropic.NewUserMessage(anthropic.NewTextBlock(userInput))
			conversation = append(conversation, userMessage)
		}

		message, err := a.runInference(ctx, conversation)
		if err != nil {
			return err
		}
		conversation = append(conversation, message.ToParam())

		toolResults := []anthropic.ContentBlockParamUnion{}
		for _, content := range message.Content {
			switch content.Type {
			case "text":
				fmt.Printf("\u001b[93mClaude\u001b[0m: %s\n", content.Text)
			case "tool_use":
				result := a.executeTool(content.ID, content.Name, content.Input)
				toolResults = append(toolResults, result)
			}
		}
		if len(toolResults) == 0 {
			readUserInput = true
			continue
		}
		readUserInput = false
		conversation = append(conversation, anthropic.NewUserMessage(toolResults...))
	}

	return nil
}

func (a *Agent) runInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
	anthropicTools := []anthropic.ToolUnionParam{}
	// Loop over the tools on the agent and convert them to Anthropic tools
	for _, tool := range a.tools {
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
	}

	message, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_7SonnetLatest,
		MaxTokens: int64(1024),
		Messages:  conversation,   // Use the current conversation (entire history)
		Tools:     anthropicTools, // Add the tools to the request
	})
	return message, err
}

func (a *Agent) executeTool(id, name string, input json.RawMessage) anthropic.ContentBlockParamUnion {
	var toolDef ToolDefinition
	var found bool
	// find the tool definition for the tool we want to execute on the agent
	for _, tool := range a.tools {
		if tool.Name == name {
			toolDef = tool
			found = true
			break
		}
	}
	if !found {
		// if the tool is not found, return a tool result block with an error message
		return anthropic.NewToolResultBlock(id, "tool not found", true)
	}

	// print the tool name and input to the console (we're calling it)
	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, input)

	// call the tool function with the input
	response, err := toolDef.Function(input)
	if err != nil {
		// if the tool function returns an error, return a tool result block with the error message
		return anthropic.NewToolResultBlock(id, err.Error(), true)
	}
	// if the tool function returns a response, return a tool result block with the response
	return anthropic.NewToolResultBlock(id, response, false)
}

type ToolDefinition struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error)
}

var ReadFileDefinition = ToolDefinition{
	Name:        "read_file",
	Description: "Read the contents of a given relative file path. Use this when you want to see what's inside a fiile. Do not use this with directory names.",
	InputSchema: ReadFileInputSchema,
	Function:    ReadFile,
}

type ReadFileInput struct {
	Path string `json:"path" jsonschema_description:"The relative path of a file in the working directory."`
}

var ReadFileInputSchema = GenerateSchema[ReadFileInput]()

func ReadFile(input json.RawMessage) (string, error) {
	readFileInput := ReadFileInput{}
	err := json.Unmarshal(input, &readFileInput)
	if err != nil {
		panic(err)
	}

	content, err := os.ReadFile(readFileInput.Path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

var ListFilesDefinition = ToolDefinition{
	Name:        "list_files",
	Description: "List files and directories at a given path. If no path is provided, list files in the current directory.",
	InputSchema: ListFilesInputSchema,
	Function:    ListFiles,
}

type ListFilesInput struct {
	Path string `json:"path,omitempty" jsonschema_description:"Optional relative path to list files from. Defaults to current directory if not provided."`
}

var ListFilesInputSchema = GenerateSchema[ListFilesInput]()

func ListFiles(input json.RawMessage) (string, error) {
	listFilesInput := ListFilesInput{}
	err := json.Unmarshal(input, &listFilesInput)
	if err != nil {
		panic(err)
	}

	dir := "."
	if listFilesInput.Path != "" {
		dir = listFilesInput.Path
	}

	var files []string
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		if relPath != "." {
			if info.IsDir() {
				files = append(files, relPath+"/")
			} else {
				files = append(files, relPath)
			}
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	result, err := json.Marshal(files)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

var EditFileDefinition = ToolDefinition{
	Name: "edit_file",
	Description: `A powerful tool for performing precise, line-based edits on a file.
Use this when you need to modify existing code or text files by replacing,
inserting, or deleting specific lines, or by replacing substrings within
a line. It accepts a list of edit operations to apply in a single,
atomic transaction. This is ideal for surgical changes rather than
overwriting entire files. All line numbers are 1-based.
If you need to create a new file, use the "append_to_file" operation as the first operation in the Edits array.
`,
	InputSchema: EditFileInputSchema,
	Function:    EditFile,
}

type StringOrStringArray []string

func (s *StringOrStringArray) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = []string{str}
		return nil
	}

	// If not string, try as a string array
	var strArr []string
	if err := json.Unmarshal(data, &strArr); err == nil {
		*s = strArr
		return nil
	}

	return fmt.Errorf("invalid input")
}

func (s StringOrStringArray) MarshalJSON() ([]byte, error) {
	if len(s) == 1 {
		return json.Marshal(s[0])
	}
	return json.Marshal([]string(s))
}

type EditOperationType string

const (
	ReplaceLineOp         EditOperationType = "replace_line"
	InsertLineBeforeOp    EditOperationType = "insert_line_before"
	InsertLineAfterOp     EditOperationType = "insert_line_after"
	DeleteLineOp          EditOperationType = "delete_line"
	ReplaceStringInLineOp EditOperationType = "replace_string_in_line"
	AppendToFileOp        EditOperationType = "append_to_file"
)

type Edit struct {
	OperationType EditOperationType   `json:"operation_type" jsonschema_description:"The type of edit to perform. There should only be one operation type per edit." jsonschema_enum:"replace_line,insert_line_before,insert_line_after,delete_line,replace_string_in_line,append_to_file"`
	LineNumber    int                 `json:"line_number,omitempty" jsonschema_description:"The 1-based line number for the operation. Required for line-specific edits." jsonschema_minimum:"1"`
	NewContent    StringOrStringArray `json:"new_content,omitempty" jsonschema_description:"The content to insert or replace with. If an array with a single string, it's treated as a single line. If an array with multiple strings, each element is a new line. Required for 'replace_line', 'insert_line_before', 'insert_line_after', 'append_to_file'."`
	OldString     string              `json:"old_string,omitempty" jsonschema_description:"The substring to find and replace. Required for 'replace_string_in_line'."`
	NewString     string              `json:"new_string,omitempty" jsonschema_description:"The string to replace 'old_string' with. Required for 'replace_string_in_line'."`
	Count         int                 `json:"count,omitempty" jsonschema_description:"Maximum number of occurrences to replace within the line for 'replace_string_in_line'. Use -1 for all occurrences. Default is 1."`
}

type EditFileInput struct {
	Path  string `json:"path" jsonschema_description:"The path to the file"`
	Edits []Edit `json:"edits" jsonschema_description:"A list of edit operations to apply to the file."`
}

var EditFileInputSchema = GenerateSchema[EditFileInput]()

func EditFile(input json.RawMessage) (string, error) {
	editFileInput := EditFileInput{}
	err := json.Unmarshal(input, &editFileInput)
	if err != nil {
		return "", err
	}

	if editFileInput.Path == "" {
		return "", fmt.Errorf("invalid input parameters")
	}

	operationCounter := 0

	fileContent, err := ReadFile(json.RawMessage(`{"path": "` + editFileInput.Path + `"}`))
	if err != nil {
		if os.IsNotExist(err) && len(editFileInput.Edits) > 0 && editFileInput.Edits[0].OperationType == AppendToFileOp {
			if len(editFileInput.Edits[0].NewContent) == 0 {
				return "", fmt.Errorf("no new content provided")
			}
			_, err = createNewFile(editFileInput.Path, editFileInput.Edits[0].NewContent[0])
			if err != nil {
				return "", err
			} else {
				operationCounter++
			}
		} else {
			return "", err
		}
	}

	lines := strings.Split(fileContent, "\n")

	for operationCounter < len(editFileInput.Edits) {
		switch editFileInput.Edits[operationCounter].OperationType {
		case ReplaceLineOp:
			err = replaceLine(&lines, editFileInput.Edits[operationCounter].LineNumber, editFileInput.Edits[operationCounter].NewContent)
			if err != nil {
				return "", err
			}
		case InsertLineBeforeOp:
			err = insertLineBefore(&lines, editFileInput.Edits[operationCounter].LineNumber, editFileInput.Edits[operationCounter].NewContent)
			if err != nil {
				return "", err
			}
		case InsertLineAfterOp:
			err = insertLineAfter(&lines, editFileInput.Edits[operationCounter].LineNumber, editFileInput.Edits[operationCounter].NewContent)
			if err != nil {
				return "", err
			}
		case DeleteLineOp:
			err = deleteLine(&lines, editFileInput.Edits[operationCounter].LineNumber)
			if err != nil {
				return "", err
			}
		case ReplaceStringInLineOp:
			if editFileInput.Edits[operationCounter].Count == 0 {
				return "", fmt.Errorf("count must be greater than 0")
			}
			err = replaceStringInLine(&lines, editFileInput.Edits[operationCounter].LineNumber, editFileInput.Edits[operationCounter].OldString, editFileInput.Edits[operationCounter].NewString, editFileInput.Edits[operationCounter].Count)
			if err != nil {
				return "", err
			}
		case AppendToFileOp:
			err = appendToFile(&lines, editFileInput.Edits[operationCounter].NewContent)
			if err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("invalid operation type: %s", editFileInput.Edits[operationCounter].OperationType)
		}
		operationCounter++
	}

	// join the lines back into a string
	linesString := strings.Join(lines, "\n")

	// rewrite the file
	err = os.WriteFile(editFileInput.Path, []byte(linesString), 0644)
	if err != nil {
		return "", err
	}

	return "OK", nil
}

func createNewFile(filePath, content string) (string, error) {
	dir := path.Dir(filePath)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	return fmt.Sprintf("Successfully created file %s", filePath), nil
}

// replaceLine replaces a line in the content with the new content
// newContent is a string or an array of strings
// if newContent is a string, it's treated as a single line
// if newContent is an array of strings, each element is a new line
func replaceLine(content *[]string, lineNumber int, newContent StringOrStringArray) error {
	if lineNumber < 1 || lineNumber > len(*content) {
		return fmt.Errorf("line number out of bounds")
	}
	index := lineNumber - 1

	result := make([]string, len(*content)+len(newContent)-1)
	// copy the content before the line number
	copy(result, (*content)[:index])
	// copy the new content over, replacing the line specified
	copy(result[index:], newContent)
	// copy the content after the line number
	copy(result[index+len(newContent):], (*content)[index+1:])
	*content = result
	return nil
}

// insertLineBefore inserts a line or lines before the specified line number, not replacing the line specified
// newContent is a string or an array of strings
// if newContent is a string, it's treated as a single line
// if newContent is an array of strings, each element is a new line
func insertLineBefore(content *[]string, lineNumber int, newContent StringOrStringArray) error {
	if lineNumber < 1 || lineNumber > len(*content) {
		return fmt.Errorf("line number out of bounds")
	}

	result := make([]string, len(*content)+len(newContent))

	// find the line number to start inserting at
	insertLine := max(lineNumber-len(newContent), 0)
	// copy the content up through the insert line
	copy(result, (*content)[:insertLine])
	// copy the new content over, inserting before the line specified
	copy(result[insertLine:], newContent)
	// copy the content after the insert line
	copy(result[insertLine+len(newContent):], (*content)[insertLine:])
	*content = result
	return nil
}

// insertLineAfter inserts a line or lines after the specified line number, not replacing the line specified
// newContent is a string or an array of strings
// if newContent is a string, it's treated as a single line
// if newContent is an array of strings, each element is a new line
func insertLineAfter(content *[]string, lineNumber int, newContent StringOrStringArray) error {
	if lineNumber < 1 || lineNumber > len(*content) {
		return fmt.Errorf("line number out of bounds")
	}
	result := make([]string, len(*content)+len(newContent))

	// copy the content up through the line number
	copy(result, (*content)[:lineNumber])
	// copy the new content over, inserting after the line specified
	copy(result[lineNumber:], newContent)
	// copy the content after the line number
	copy(result[lineNumber+len(newContent):], (*content)[lineNumber:])
	*content = result
	return nil
}

// deleteLine deletes a line from the content
// lineNumber is the 1-based line number to delete
func deleteLine(content *[]string, lineNumber int) error {
	if lineNumber < 1 || lineNumber > len(*content) {
		return fmt.Errorf("line number out of bounds")
	}

	lines := *content
	lines = slices.Delete(lines, lineNumber-1, lineNumber)
	*content = lines
	return nil
}

// replaceStringInLine replaces a string in a line with a new string
// lineNumber is the 1-based line number to replace the string in
// oldString is the string to replace
// newString is the string to replace oldString with
// count is the maximum number of occurrences to replace, defaults to 1, -1 for all occurrences
func replaceStringInLine(content *[]string, lineNumber int, oldString, newString string, count int) error {
	if lineNumber < 1 || lineNumber > len(*content) {
		return fmt.Errorf("line number out of bounds")
	}

	lines := *content
	newLine := strings.Replace(lines[lineNumber-1], oldString, newString, count)
	lines[lineNumber-1] = newLine
	*content = lines
	return nil
}

// appendToFile appends content to the end of the file
// content is a string or an array of strings
// if content is a string, it's treated as a single line
// if content is an array of strings, each element is a new line
func appendToFile(content *[]string, newContent StringOrStringArray) error {
	*content = append(*content, newContent...)
	return nil
}

var ReadLinesDefinition = ToolDefinition{
	Name: "read_lines",
	Description: `Read lines from a file.

	Reads lines from a file starting at line 'start_line' and ending at line 'end_line', with 'start_line' inclusive and 'end_line' exclusive.

	"start_line" and "end_line" are 1-indexed.

	If 'start_line' is greater than 'end_line', return an error.

	If 'start_line' or 'end_line' is greater than the number of lines in the file, return an error.

	If the file doesn't exist, return an error.
	`,
	InputSchema: ReadLinesInputSchema,
	Function:    ReadLines,
}

type ReadLinesInput struct {
	Path      string `json:"path" jsonschema_description:"The path to the file"`
	StartLine int    `json:"start_line" jsonschema_description:"The line to start reading from (1-indexed), inclusive"`
	EndLine   int    `json:"end_line" jsonschema_description:"The line to stop reading at (1-indexed), exclusive"`
}

var ReadLinesInputSchema = GenerateSchema[ReadLinesInput]()

func ReadLines(input json.RawMessage) (string, error) {
	readLinesInput := ReadLinesInput{}
	err := json.Unmarshal(input, &readLinesInput)
	if err != nil {
		return "", err
	}

	if readLinesInput.StartLine < 1 || readLinesInput.EndLine < readLinesInput.StartLine {
		return "", fmt.Errorf("invalid line numbers")
	}

	if readLinesInput.StartLine == readLinesInput.EndLine {
		return "[]", nil
	}

	file, err := os.Open(readLinesInput.Path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	lines := []string{}
	scanner := bufio.NewScanner(file)
	lineNum := 1
	for scanner.Scan() {
		if lineNum >= readLinesInput.StartLine && lineNum < readLinesInput.EndLine {
			line := fmt.Sprintf("<line-%d>%s</line-%d>", lineNum, scanner.Text(), lineNum)
			lines = append(lines, line)
		}
		lineNum++
	}

	if lineNum <= readLinesInput.StartLine {
		return "", fmt.Errorf("start line beyond file length")
	}

	if lineNum < readLinesInput.EndLine {
		return "", fmt.Errorf("end line beyond file length")
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	result, err := json.Marshal(lines)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func GenerateSchema[T any]() anthropic.ToolInputSchemaParam {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T

	schema := reflector.Reflect(v)

	return anthropic.ToolInputSchemaParam{
		Properties: schema.Properties,
	}
}
