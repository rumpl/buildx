package progress

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/moby/buildkit/client"
	"github.com/opencontainers/go-digest"
	"github.com/rivo/tview"
)

const width = 60

type tuiWriter struct {
	app      *tview.Application
	tree     *tview.TreeView
	logsView *tview.TextView
	vertices map[digest.Digest]*tview.TreeNode
	logs     map[digest.Digest]string
}

func newTUIWriter() Writer {
	app := tview.NewApplication()
	rootDir := "Build stages"
	root := tview.NewTreeNode(rootDir).
		SetColor(tcell.ColorRed)
	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)

	grid := tview.NewGrid().SetRows(1).SetColumns(width, 0).SetBorders(true)
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	grid.AddItem(tree, 1, 0, 1, 1, 0, 0, false).AddItem(textView, 1, 1, 1, 1, 0, 0, false)

	app.SetRoot(grid, true).EnableMouse(true)

	go func() {
		if err := app.Run(); err != nil {
			panic(err)
		}
	}()
	tw := &tuiWriter{
		app:      app,
		tree:     tree,
		vertices: map[digest.Digest]*tview.TreeNode{},
		logs:     map[digest.Digest]string{},
		logsView: textView,
	}
	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		d := node.GetReference()
		if d == nil {
			return
		}
		if lw, ok := tw.logs[d.(digest.Digest)]; ok {
			if lw != "" {
				tw.logsView.SetText(lw)
			} else {
				tw.logsView.SetText("No logs...")
			}
		} else {
			tw.logsView.SetText("No logs...")
		}
	})

	return tw
}

// ClearLogSource implements Writer
func (t *tuiWriter) ClearLogSource(interface{}) {
}

// ValidateLogSource implements Writer
func (t *tuiWriter) ValidateLogSource(digest.Digest, interface{}) bool {
	return true
}

// Wait implements Writer
func (t *tuiWriter) Wait() error {
	time.Sleep(10 * time.Second)
	t.app.Stop()
	return nil
}

// Warnings implements Writer
func (t *tuiWriter) Warnings() []client.VertexWarning {
	return []client.VertexWarning{}
}

// Write implements Writer
func (t *tuiWriter) Write(status *client.SolveStatus) {
	t.app.QueueUpdateDraw(func() {
		for _, v := range status.Vertexes {
			name := strings.Replace(v.Name, "\t", "", -1)
			if v.Cached {
				name = "CACHED " + name
			}
			if len(name) > width {
				name = strings.Replace(name[:width], "\t", "", -1)
			}

			if a, ok := t.vertices[v.Digest]; ok {
				if v.Completed != nil {
					a.SetText(name)
					a.SetColor(tcell.ColorBlue)
				}
				if v.Error != "" {
					a.SetColor(tcell.ColorRed)
				}
			} else {
				node := tview.NewTreeNode(name).SetReference(v.Digest)
				if v.Completed != nil {
					node = node.SetColor(tcell.ColorBlue)
				}

				t.tree.GetRoot().AddChild(node)
				t.vertices[v.Digest] = node
			}
		}
		for _, l := range status.Logs {
			if _, ok := t.logs[l.Vertex]; !ok {
				t.logs[l.Vertex] = ""
			}
			t.logs[l.Vertex] += string(l.Data)
		}
	})
}
