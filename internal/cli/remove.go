package cli

import (
	"github.com/spf13/cobra"
	"github.com/thobiassilva/wt/internal/worktree"
)

func buildRemoveCmd(svc *worktree.Service, out *Output) *cobra.Command {
	var (
		force        bool
		deleteBranch bool
		dryRun       bool
	)

	cmd := &cobra.Command{
		Use:     "remove <worktree>",
		Aliases: []string{"rm"},
		Short:   "Remove uma worktree",
		Long: `remove - Remove uma worktree

O <worktree> pode ser o nome da branch, o nome do diretorio derivado, ou o
caminho da worktree. Por padrao a branch git associada e mantida; use
--delete-branch para remove-la junto.`,
		Example: `  wt remove feature/loginForm
  wt remove feature-login-form
  wt remove feature/loginForm --delete-branch
  wt remove feature/loginForm --force
  wt remove feature/loginForm --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wt, err := svc.ResolveWorktree(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			if dryRun {
				out.Info("[dry-run] Removeria worktree: %s", wt.Path)
				if deleteBranch && wt.Branch != "" {
					out.Info("[dry-run] Removeria branch: %s", wt.Branch)
				}
				out.Info("[dry-run] Nenhuma alteracao foi feita.")
				return nil
			}

			if err := svc.RemoveWorktree(cmd.Context(), wt, force, deleteBranch); err != nil {
				return err
			}

			out.Info("Worktree removida: %s", wt.Path)
			if deleteBranch && wt.Branch != "" {
				out.Info("Branch removida: %s", wt.Branch)
			}
			return nil
		},
		SilenceUsage: true,
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Forca a remocao (repassa --force ao git)")
	cmd.Flags().BoolVarP(&deleteBranch, "delete-branch", "D", false, "Remove tambem a branch git associada")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Mostra o que seria feito sem executar")

	return cmd
}
