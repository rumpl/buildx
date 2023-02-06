package progress

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/moby/buildkit/client"
	"github.com/opencontainers/go-digest"
	"github.com/rivo/tview"
)

const width = 160

type tuiWriter struct {
	app      *tview.Application
	tree     *tview.TreeView
	vertices map[digest.Digest]*tview.TreeNode
}

func newTUIWriter() Writer {
	newPrimitive := func(text string) tview.Primitive {
		return tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText(text)
	}

	// menu := newPrimitive("Stages")
	main := newPrimitive("Logs")

	rootDir := "."
	root := tview.NewTreeNode(rootDir).
		SetColor(tcell.ColorRed)
	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)

	// main.AddItemDir(tree)
	grid := tview.NewGrid().SetRows(1).SetColumns(width, 0).SetBorders(true)

	grid.AddItem(tree, 1, 0, 1, 1, 0, 0, false).AddItem(main, 1, 1, 1, 1, 0, 0, false)

	app := tview.NewApplication().SetRoot(grid, true).EnableMouse(true)

	go func() {
		if err := app.Run(); err != nil {
			panic(err)
		}
	}()

	return &tuiWriter{
		app:      app,
		tree:     tree,
		vertices: map[digest.Digest]*tview.TreeNode{},
	}
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
				node := tview.NewTreeNode(name)
				if v.Completed != nil {
					node = node.SetColor(tcell.ColorBlue)
				}

				t.tree.GetRoot().AddChild(node)
				t.vertices[v.Digest] = node
			}
		}
	})
}
