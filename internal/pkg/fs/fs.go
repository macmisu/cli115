package fs

import (
	"github.com/deadblue/elevengo"
	"go.dead.blue/cli115/internal/pkg/util"
	"strings"
)

/*
A file system wrapper for all file operations on cloud storage.
*/
type RemoteFs struct {
	agent *elevengo.Agent
	// root dir
	root *DirNode
	// current dir
	curr *DirNode
	// files cache under current dir
	files map[string]*elevengo.File
}

func (f *RemoteFs) Root() *DirNode {
	return f.root
}

func (f *RemoteFs) Curr() *DirNode {
	return f.curr
}

func (f *RemoteFs) SetCurr(dir *DirNode) {
	f.curr = dir
	// Update file cache
	f.files = make(map[string]*elevengo.File)
	for cur := elevengo.FileCursor(); cur.HasMore(); cur.Next() {
		if files, err := f.agent.FileList(f.curr.Id, cur); err != nil {
			break
		} else {
			for _, file := range files {
				if file.IsFile {
					f.files[file.Name] = file
				}
				if file.IsDirectory {
					dir.Append(file.FileId, file.Name)
				}
			}
		}
	}
}

/*
Locate directory by path.
"path" starts with "/" means an absolute path, otherwise a relative path.
*/
func (f *RemoteFs) LocateDir(path string) (dir *DirNode) {
	dir = f.curr
	dirs := strings.Split(path, "/")
	depth, start := len(dirs), 0
	if depth > 1 && dirs[0] == "" {
		// Starts from root
		dir = f.root
		start = 1
	}
	// Go deep
	for i := start; i < depth; i += 1 {
		dirName := util.StdUnescape(dirs[i])
		if dirName == "." || dirName == "" {
			// "." means current dir
			continue
		} else if dirName == ".." {
			// ".." means upstairs dir
			if dir != f.root {
				dir = dir.Parent
			}
		} else {
			if !dir.ChildrenCached {
				f.fetchChildren(dir)
			}
			dir = dir.Children[dirName]
		}
		if dir == nil {
			break
		}
	}
	return dir
}

func (f *RemoteFs) fetchChildren(dir *DirNode) {
	for cur := elevengo.FileCursor(); cur.HasMore(); cur.Next() {
		if files, err := f.agent.FileList(dir.Id, cur); err != nil {
			break
		} else {
			for _, file := range files {
				if !file.IsDirectory {
					continue
				}
				dir.Append(file.FileId, file.Name)
			}
		}
	}
	dir.ChildrenCached = true
}

// Get a file from current directory with specific
// name, or return nil when not found.
func (f *RemoteFs) File(name string) *elevengo.File {
	return f.files[util.StdUnescape(name)]
}

// Get files from current directory whose name matches the given pattern.
func (f *RemoteFs) Files(pattern string) []*elevengo.File {
	result := make([]*elevengo.File, 0)
	for name, file := range f.files {
		if MustMatch(pattern, name) {
			result = append(result, file)
		}
	}
	return result
}

func (f *RemoteFs) DirNames(dir *DirNode, prefix string) (names []string) {
	prefix, names = util.StdUnescape(prefix), make([]string, 0)
	if dir == nil {
		dir = f.curr
	}
	if !dir.ChildrenCached {
		f.fetchChildren(dir)
	}
	for name := range dir.Children {
		if prefix == "" || strings.HasPrefix(name, prefix) {
			names = append(names, escape(name)+"/")
		}
	}
	return
}

func (f *RemoteFs) FileNames(prefix string) (names []string) {
	prefix, names = util.StdUnescape(prefix), make([]string, 0)
	for name := range f.files {
		if prefix == "" || strings.HasPrefix(name, prefix) {
			names = append(names, escape(name))
		}
	}
	return names
}

func New(agent *elevengo.Agent) *RemoteFs {
	root := MakeNode("0", "")
	fs := &RemoteFs{
		agent: agent,
		root:  root,
		files: make(map[string]*elevengo.File),
	}
	fs.SetCurr(root)
	return fs
}
