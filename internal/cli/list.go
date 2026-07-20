package cli

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thobiassilva/wt/internal/gitx"
	"github.com/thobiassilva/wt/internal/worktree"
)

func buildListCmd(svc *worktree.Service, out *Output) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lista as worktrees do repositorio",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			worktrees, current, err := svc.List(cmd.Context())
			if err != nil {
				return err
			}
			out.Table(
				[]string{"", "NOME", "BRANCH", "CAMINHO"},
				listRows(worktrees, current),
			)
			return nil
		},
		SilenceUsage: true,
	}
}

// listRows builds the table rows for `wt list`. The current worktree is marked
// with "*" in the first column.
func listRows(worktrees []gitx.Worktree, current string) [][]string {
	rows := make([][]string, 0, len(worktrees))
	for _, wt := range worktrees {
		marker := ""
		if sameWorktreePath(wt.Path, current) {
			marker = "*"
		}
		rows = append(rows, []string{marker, filepath.Base(wt.Path), branchLabel(wt), wt.Path})
	}
	return rows
}

// branchLabel renders the branch column, handling detached/bare worktrees.
func branchLabel(wt gitx.Worktree) string {
	switch {
	case wt.Bare:
		return "(bare)"
	case wt.Detached:
		return "(detached)"
	case wt.Branch == "":
		return "-"
	default:
		return wt.Branch
	}
}

// sameWorktreePath compares two paths for equality, resolving symlinks so that
// e.g. macOS /tmp and /private/tmp match.
func sameWorktreePath(a, b string) bool {
	if a == b {
		return true
	}
	ra, err := filepath.EvalSymlinks(a)
	if err != nil {
		ra = filepath.Clean(a)
	}
	rb, err := filepath.EvalSymlinks(b)
	if err != nil {
		rb = filepath.Clean(b)
	}
	return ra == rb
}
