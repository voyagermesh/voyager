package volume

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
)

func TestNewAtomicWriter(t *testing.T) {
	targetDir, err := ioutil.TempDir(os.TempDir(), "atomic-write")
	if err != nil {
		t.Fatalf("unexpected error creating tmp dir: %v", err)
	}

	_, err = NewAtomicWriter(targetDir)
	if err != nil {
		t.Fatalf("unexpected error creating writer for existing target dir: %v", err)
	}

	nonExistentDir, err := ioutil.TempDir(os.TempDir(), "atomic-write")
	if err != nil {
		t.Fatalf("unexpected error creating tmp dir: %v", err)
	}
	err = os.Remove(nonExistentDir)
	if err != nil {
		t.Fatalf("unexpected error ensuring dir %v does not exist: %v", nonExistentDir, err)
	}

	_, err = NewAtomicWriter(nonExistentDir)
	if err == nil {
		t.Fatalf("unexpected success creating writer for nonexistent target dir: %v", err)
	}
}

func TestValidatePath(t *testing.T) {
	maxPath := strings.Repeat("a", maxPathLength+1)
	maxFile := strings.Repeat("a", maxFileNameLength+1)

	cases := []struct {
		name  string
		path  string
		valid bool
	}{
		{
			name:  "valid 1",
			path:  "i/am/well/behaved.txt",
			valid: true,
		},
		{
			name:  "valid 2",
			path:  "keepyourheaddownandfollowtherules.txt",
			valid: true,
		},
		{
			name:  "max path length",
			path:  maxPath,
			valid: false,
		},
		{
			name:  "max file length",
			path:  maxFile,
			valid: false,
		},
		{
			name:  "absolute failure",
			path:  "/dev/null",
			valid: false,
		},
		{
			name:  "reserved path",
			path:  "..sneaky.txt",
			valid: false,
		},
		{
			name:  "contains doubledot 1",
			path:  "hello/there/../../../../../../etc/passwd",
			valid: false,
		},
		{
			name:  "contains doubledot 2",
			path:  "hello/../etc/somethingbad",
			valid: false,
		},
		{
			name:  "empty",
			path:  "",
			valid: false,
		},
	}

	for _, tc := range cases {
		err := validatePath(tc.path)
		if tc.valid && err != nil {
			t.Errorf("%v: unexpected failure: %v", tc.name, err)
			continue
		}

		if !tc.valid && err == nil {
			t.Errorf("%v: unexpected success", tc.name)
		}
	}
}

func TestWriteFile(t *testing.T) {
	atomicWriter := &AtomicWriter{targetDir: os.TempDir() + "/atomic-writer"}
	fileMap := map[string]FileProjection{
		"file.txt": {Data: []byte("hello-world"), Mode: 0777},
		"configs":  {Data: []byte("hello from configs"), Mode: 0777},
	}
	atomicWriter.Write(fileMap)

	for k, v := range fileMap {
		data, err := ioutil.ReadFile(os.TempDir() + "/atomic-writer/" + k)
		if err != nil {
			t.Fatalf("Expected nil, got error: %v", err)
		}
		if !reflect.DeepEqual(data, v.Data) {
			t.Fatalf("Data not equal for file %s", k)
		}
		os.Remove(os.TempDir() + "/atomic-writer/" + k)
	}
}

func TestWriteOnce(t *testing.T) {
	// $1 if you can tell me what this binary is
	encodedMysteryBinary := `f0VMRgIBAQAAAAAAAAAAAAIAPgABAAAAeABAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAEAAOAAB
AAAAAAAAAAEAAAAFAAAAAAAAAAAAAAAAAEAAAAAAAAAAQAAAAAAAfQAAAAAAAAB9AAAAAAAAAAAA
IAAAAAAAsDyZDwU=`

	mysteryBinaryBytes := make([]byte, base64.StdEncoding.DecodedLen(len(encodedMysteryBinary)))
	numBytes, err := base64.StdEncoding.Decode(mysteryBinaryBytes, []byte(encodedMysteryBinary))
	if err != nil {
		t.Fatalf("Unexpected error decoding binary payload: %v", err)
	}

	if numBytes != 125 {
		t.Fatalf("Unexpected decoded binary size: expected 125, got %v", numBytes)
	}

	cases := []struct {
		name    string
		payload map[string]FileProjection
		success bool
	}{
		{
			name: "invalid payload 1",
			payload: map[string]FileProjection{
				"foo":        {Mode: 0644, Data: []byte("foo")},
				"..bar":      {Mode: 0644, Data: []byte("bar")},
				"binary.bin": {Mode: 0644, Data: mysteryBinaryBytes},
			},
			success: false,
		},
		{
			name: "invalid payload 2",
			payload: map[string]FileProjection{
				"foo/../bar": {Mode: 0644, Data: []byte("foo")},
			},
			success: false,
		},
		{
			name: "basic 1",
			payload: map[string]FileProjection{
				"foo": {Mode: 0644, Data: []byte("foo")},
				"bar": {Mode: 0644, Data: []byte("bar")},
			},
			success: true,
		},
		{
			name: "basic 2",
			payload: map[string]FileProjection{
				"binary.bin":  {Mode: 0644, Data: mysteryBinaryBytes},
				".binary.bin": {Mode: 0644, Data: mysteryBinaryBytes},
			},
			success: true,
		},
		{
			name: "basic mode 1",
			payload: map[string]FileProjection{
				"foo": {Mode: 0777, Data: []byte("foo")},
				"bar": {Mode: 0400, Data: []byte("bar")},
			},
			success: true,
		},
		{
			name: "dotfiles",
			payload: map[string]FileProjection{
				"foo":           {Mode: 0644, Data: []byte("foo")},
				"bar":           {Mode: 0644, Data: []byte("bar")},
				".dotfile":      {Mode: 0644, Data: []byte("dotfile")},
				".dotfile.file": {Mode: 0644, Data: []byte("dotfile.file")},
			},
			success: true,
		},
		{
			name: "dotfiles mode",
			payload: map[string]FileProjection{
				"foo":           {Mode: 0407, Data: []byte("foo")},
				"bar":           {Mode: 0440, Data: []byte("bar")},
				".dotfile":      {Mode: 0777, Data: []byte("dotfile")},
				".dotfile.file": {Mode: 0666, Data: []byte("dotfile.file")},
			},
			success: true,
		},
		{
			name: "subdirectories 1",
			payload: map[string]FileProjection{
				"foo/bar.txt": {Mode: 0644, Data: []byte("foo/bar")},
				"bar/zab.txt": {Mode: 0644, Data: []byte("bar/zab.txt")},
			},
			success: true,
		},
		{
			name: "subdirectories mode 1",
			payload: map[string]FileProjection{
				"foo/bar.txt": {Mode: 0400, Data: []byte("foo/bar")},
				"bar/zab.txt": {Mode: 0644, Data: []byte("bar/zab.txt")},
			},
			success: true,
		},
		{
			name: "subdirectories 2",
			payload: map[string]FileProjection{
				"foo//bar.txt":      {Mode: 0644, Data: []byte("foo//bar")},
				"bar///bar/zab.txt": {Mode: 0644, Data: []byte("bar/../bar/zab.txt")},
			},
			success: true,
		},
		{
			name: "subdirectories 3",
			payload: map[string]FileProjection{
				"foo/bar.txt":      {Mode: 0644, Data: []byte("foo/bar")},
				"bar/zab.txt":      {Mode: 0644, Data: []byte("bar/zab.txt")},
				"foo/blaz/bar.txt": {Mode: 0644, Data: []byte("foo/blaz/bar")},
				"bar/zib/zab.txt":  {Mode: 0644, Data: []byte("bar/zib/zab.txt")},
			},
			success: true,
		},
		{
			name: "kitchen sink",
			payload: map[string]FileProjection{
				"foo.log":                           {Mode: 0644, Data: []byte("foo")},
				"bar.zap":                           {Mode: 0644, Data: []byte("bar")},
				".dotfile":                          {Mode: 0644, Data: []byte("dotfile")},
				"foo/bar.txt":                       {Mode: 0644, Data: []byte("foo/bar")},
				"bar/zab.txt":                       {Mode: 0644, Data: []byte("bar/zab.txt")},
				"foo/blaz/bar.txt":                  {Mode: 0644, Data: []byte("foo/blaz/bar")},
				"bar/zib/zab.txt":                   {Mode: 0400, Data: []byte("bar/zib/zab.txt")},
				"1/2/3/4/5/6/7/8/9/10/.dotfile.lib": {Mode: 0777, Data: []byte("1-2-3-dotfile")},
			},
			success: true,
		},
	}

	for _, tc := range cases {
		targetDir, err := ioutil.TempDir(os.TempDir(), "atomic-write")
		if err != nil {
			t.Errorf("%v: unexpected error creating tmp dir: %v", tc.name, err)
			continue
		}

		writer := &AtomicWriter{targetDir: targetDir}
		err = writer.Write(tc.payload)
		if err != nil && tc.success {
			t.Errorf("%v: unexpected error writing payload: %v", tc.name, err)
			continue
		} else if err == nil && !tc.success {
			t.Errorf("%v: unexpected success", tc.name)
			continue
		} else if err != nil {
			continue
		}

		checkVolumeContents(targetDir, tc.name, tc.payload, t)
	}
}

func TestUpdate(t *testing.T) {
	cases := []struct {
		name        string
		first       map[string]FileProjection
		next        map[string]FileProjection
		shouldWrite bool
	}{
		{
			name: "update",
			first: map[string]FileProjection{
				"foo": {Mode: 0644, Data: []byte("foo")},
				"bar": {Mode: 0644, Data: []byte("bar")},
			},
			next: map[string]FileProjection{
				"foo": {Mode: 0644, Data: []byte("foo2")},
				"bar": {Mode: 0640, Data: []byte("bar2")},
			},
			shouldWrite: true,
		},
		{
			name: "no update",
			first: map[string]FileProjection{
				"foo": {Mode: 0644, Data: []byte("foo")},
				"bar": {Mode: 0644, Data: []byte("bar")},
			},
			next: map[string]FileProjection{
				"foo": {Mode: 0644, Data: []byte("foo")},
				"bar": {Mode: 0644, Data: []byte("bar")},
			},
			shouldWrite: false,
		},
		{
			name: "no update 2",
			first: map[string]FileProjection{
				"foo/bar.txt": {Mode: 0644, Data: []byte("foo")},
				"bar/zab.txt": {Mode: 0644, Data: []byte("bar")},
			},
			next: map[string]FileProjection{
				"foo/bar.txt": {Mode: 0644, Data: []byte("foo")},
				"bar/zab.txt": {Mode: 0644, Data: []byte("bar")},
			},
			shouldWrite: false,
		},
		{
			name: "add 1",
			first: map[string]FileProjection{
				"foo/bar.txt": {Mode: 0644, Data: []byte("foo")},
				"bar/zab.txt": {Mode: 0644, Data: []byte("bar")},
			},
			next: map[string]FileProjection{
				"foo/bar.txt": {Mode: 0644, Data: []byte("foo")},
				"bar/zab.txt": {Mode: 0644, Data: []byte("bar")},
				"blu/zip.txt": {Mode: 0644, Data: []byte("zip")},
			},
			shouldWrite: true,
		},
		{
			name: "add 2",
			first: map[string]FileProjection{
				"foo/bar.txt": {Mode: 0644, Data: []byte("foo")},
				"bar/zab.txt": {Mode: 0644, Data: []byte("bar")},
			},
			next: map[string]FileProjection{
				"foo/bar.txt":             {Mode: 0644, Data: []byte("foo")},
				"bar/zab.txt":             {Mode: 0644, Data: []byte("bar")},
				"blu/two/2/3/4/5/zip.txt": {Mode: 0644, Data: []byte("zip")},
			},
			shouldWrite: true,
		},
		{
			name: "add 3",
			first: map[string]FileProjection{
				"foo/bar.txt": {Mode: 0644, Data: []byte("foo")},
				"bar/zab.txt": {Mode: 0644, Data: []byte("bar")},
			},
			next: map[string]FileProjection{
				"foo/bar.txt":         {Mode: 0644, Data: []byte("foo")},
				"bar/zab.txt":         {Mode: 0644, Data: []byte("bar")},
				"bar/2/3/4/5/zip.txt": {Mode: 0644, Data: []byte("zip")},
			},
			shouldWrite: true,
		},
		{
			name: "delete 1",
			first: map[string]FileProjection{
				"foo/bar.txt": {Mode: 0644, Data: []byte("foo")},
				"bar/zab.txt": {Mode: 0644, Data: []byte("bar")},
			},
			next: map[string]FileProjection{
				"foo/bar.txt": {Mode: 0644, Data: []byte("foo")},
			},
			shouldWrite: true,
		},
		{
			name: "delete 2",
			first: map[string]FileProjection{
				"foo/bar.txt":       {Mode: 0644, Data: []byte("foo")},
				"bar/1/2/3/zab.txt": {Mode: 0644, Data: []byte("bar")},
			},
			next: map[string]FileProjection{
				"foo/bar.txt": {Mode: 0644, Data: []byte("foo")},
			},
			shouldWrite: true,
		},
		{
			name: "delete 3",
			first: map[string]FileProjection{
				"foo/bar.txt":       {Mode: 0644, Data: []byte("foo")},
				"bar/1/2/sip.txt":   {Mode: 0644, Data: []byte("sip")},
				"bar/1/2/3/zab.txt": {Mode: 0644, Data: []byte("bar")},
			},
			next: map[string]FileProjection{
				"foo/bar.txt":     {Mode: 0644, Data: []byte("foo")},
				"bar/1/2/sip.txt": {Mode: 0644, Data: []byte("sip")},
			},
			shouldWrite: true,
		},
		{
			name: "delete 4",
			first: map[string]FileProjection{
				"foo/bar.txt":            {Mode: 0644, Data: []byte("foo")},
				"bar/1/2/sip.txt":        {Mode: 0644, Data: []byte("sip")},
				"bar/1/2/3/4/5/6zab.txt": {Mode: 0644, Data: []byte("bar")},
			},
			next: map[string]FileProjection{
				"foo/bar.txt":     {Mode: 0644, Data: []byte("foo")},
				"bar/1/2/sip.txt": {Mode: 0644, Data: []byte("sip")},
			},
			shouldWrite: true,
		},
		{
			name: "delete all",
			first: map[string]FileProjection{
				"foo/bar.txt":            {Mode: 0644, Data: []byte("foo")},
				"bar/1/2/sip.txt":        {Mode: 0644, Data: []byte("sip")},
				"bar/1/2/3/4/5/6zab.txt": {Mode: 0644, Data: []byte("bar")},
			},
			next:        map[string]FileProjection{},
			shouldWrite: true,
		},
		{
			name: "add and delete 1",
			first: map[string]FileProjection{
				"foo/bar.txt": {Mode: 0644, Data: []byte("foo")},
			},
			next: map[string]FileProjection{
				"bar/baz.txt": {Mode: 0644, Data: []byte("baz")},
			},
			shouldWrite: true,
		},
	}

	for _, tc := range cases {
		targetDir, err := ioutil.TempDir(os.TempDir(), "atomic-write")
		if err != nil {
			t.Errorf("%v: unexpected error creating tmp dir: %v", tc.name, err)
			continue
		}

		writer := &AtomicWriter{targetDir: targetDir}

		err = writer.Write(tc.first)
		if err != nil {
			t.Errorf("%v: unexpected error writing: %v", tc.name, err)
			continue
		}

		checkVolumeContents(targetDir, tc.name, tc.first, t)
		if !tc.shouldWrite {
			continue
		}

		err = writer.Write(tc.next)
		if err != nil {
			if tc.shouldWrite {
				t.Errorf("%v: unexpected error writing: %v", tc.name, err)
				continue
			}
		} else if !tc.shouldWrite {
			t.Errorf("%v: unexpected success", tc.name)
			continue
		}

		checkVolumeContents(targetDir, tc.name, tc.next, t)
	}
}

func checkVolumeContents(targetDir, tcName string, payload map[string]FileProjection, t *testing.T) {
	// use filepath.Walk to reconstruct the payload, then deep equal
	observedPayload := make(map[string]FileProjection)
	visitor := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		relativePath := strings.TrimPrefix(path, targetDir)
		relativePath = strings.TrimPrefix(relativePath, "/")
		if strings.HasPrefix(relativePath, "..") {
			return nil
		}
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		fileInfo, err := os.Stat(path)
		if err != nil {
			return err
		}
		mode := int32(fileInfo.Mode())

		observedPayload[relativePath] = FileProjection{Data: content, Mode: mode}

		return nil
	}

	err := filepath.Walk(targetDir, visitor)
	if err != nil {
		t.Errorf("%v: unexpected error walking directory: %v", tcName, err)
	}

	cleanPathPayload := make(map[string]FileProjection, len(payload))
	for k, v := range payload {
		cleanPathPayload[path.Clean(k)] = v
	}
	if !reflect.DeepEqual(cleanPathPayload, observedPayload) {
		t.Errorf("%v: payload and observed payload do not match.", tcName)
		t.Errorf("Expected %s;\nGot %s\n", cleanPathPayload, observedPayload)
		t.Errorf("%s", string(debug.Stack()))
	}
}
