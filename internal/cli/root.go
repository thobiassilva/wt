// Package cli wires cobra commands and flags to the worktree domain.
package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/thobiassilva/wt/internal/fsx"
	"github.com/thobiassilva/wt/internal/gitx"
	"github.com/thobiassilva/wt/internal/worktree"
)

// Execute is the CLI entry point called from main.go.
func Execute(version string) error {
	out := NewOutput()
	git := gitx.New()
	fs := fsx.New()
	svc := worktree.NewService(git, fs)

	root := buildRoot(version, svc, out)
	return root.Execute()
}

func buildRoot(version string, svc *worktree.Service, out *Output) *cobra.Command {
	var (
		name      string
		base      string
		path      string
		noInclude bool
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:     "wt <branch>",
		Short:   "Gerenciador de Git Worktrees",
		Version: version,
		Args:    cobra.ExactArgs(1),
		Long: `wt - Gerenciador de Git Worktrees

Cria uma branch e uma worktree git no diretorio derivado, copiando
os arquivos listados em .worktreeinclude (se existir).

DERIVACAO (branch -> worktree):
    feature/loginForm    -> feature-login-form
    bugfix/fixApiTimeout -> bugfix-fix-api-timeout
    hotfix-urgent        -> hotfix-urgent`,
		Example: `  wt feature/loginForm
  wt feature/loginForm --name meu-fix
  wt feature/loginForm --base main
  wt feature/loginForm --path ./branchs
  wt feature/loginForm --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), svc, out, args[0], worktree.Options{
				Branch:     args[0],
				Name:       name,
				Base:       base,
				PathPrefix: path,
				NoInclude:  noInclude,
				DryRun:     dryRun,
			})
		},
		SilenceUsage: true,
	}

	cmd.Flags().StringVarP(&name, "name", "w", "", "Nome da worktree (default: derivado da branch)")
	cmd.Flags().StringVarP(&base, "base", "b", "", "Branch de origem (default: branch atual)")
	cmd.Flags().StringVarP(&path, "path", "p", "..", "Diretorio pai da worktree")
	cmd.Flags().BoolVar(&noInclude, "no-include", false, "Pular copia de .worktreeinclude")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Mostra o que seria feito sem executar")

	// Use -V for version to match bash CLI.
	cmd.Flags().BoolP("version", "V", false, "Mostra versao")
	// Bind the -V flag to cobra's built-in version handling.
	cmd.SetVersionTemplate("wt {{.Version}}\n")

	return cmd
}

func run(ctx context.Context, svc *worktree.Service, out *Output, branch string, opts worktree.Options) error {
	plan, err := svc.BuildPlan(ctx, opts)
	if err != nil {
		return err
	}

	printSummary(out, plan)

	if opts.DryRun {
		for _, f := range plan.FilesToCopy {
			out.Info("[dry-run] Copiaria: %s", f)
		}
		out.Info("[dry-run] Nenhuma alteracao foi feita.")
		return nil
	}

	if err := svc.Execute(ctx, plan); err != nil {
		return err
	}

	if len(plan.FilesToCopy) > 0 {
		out.Info("Copiados %d arquivo(s) via .worktreeinclude", len(plan.FilesToCopy))
	}

	out.Info("Worktree criada em: %s", plan.Dest)
	return nil
}

func printSummary(out *Output, plan worktree.Plan) {
	action := "Criando"
	if plan.BranchExists {
		action = "Reusando"
	}
	out.Info("%s worktree", action)
	out.Info("  Branch  : %s", plan.Branch)
	out.Info("  Base    : %s", plan.Base)
	out.Info("  Destino : %s", plan.Dest)
}
