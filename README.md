# wt

CLI para criacao de git worktrees com copia automatica de arquivos gitignored via `.worktreeinclude`.

## O que faz

- Cria uma git worktree a partir de uma branch
- Deriva automaticamente o nome da worktree a partir da branch (camelCase para kebab-case)
- Copia arquivos listados em `.worktreeinclude` para a nova worktree (ex: `.env`, chaves, configs locais)
- Suporta a sintaxe completa do `.gitignore`, incluindo **padroes de negacao** (`!`)
- Funciona em macOS, Linux e Windows

## Instalacao

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/thobiassilva/wt/main/install.sh | bash
```

O script detecta seu OS e arquitetura, baixa o binario correto do GitHub Releases e instala em `~/.local/bin/wt`.

**Windows:**

```powershell
scoop bucket add thobiassilva https://github.com/thobiassilva/scoop-bucket
scoop install wt
```

**macOS (Homebrew):**

```bash
brew install thobiassilva/tap/wt
```

Se `~/.local/bin` nao estiver no seu PATH, adicione ao seu `~/.zshrc` (ou `~/.bashrc`):

```bash
export PATH="$HOME/.local/bin:$PATH"
```

### Atualizacao

Re-execute o comando de instalacao. O script baixa a versao mais recente e substitui o binario.

### Desinstalacao

```bash
rm ~/.local/bin/wt
```

## Uso

```bash
wt <branch> [opcoes]
```

### Argumentos e opcoes

| Argumento/Opcao | Curto | Descricao | Default |
|---|---|---|---|
| `<branch>` | | Nome da branch git | obrigatorio |
| `--name` | `-w` | Nome da worktree (diretorio) | derivado da branch |
| `--base` | `-b` | Branch de origem | branch atual |
| `--path` | `-p` | Diretorio pai da worktree | `../` |
| `--no-include` | | Pular copia de `.worktreeinclude` | false |
| `--dry-run` | `-n` | Mostra o que seria feito sem executar | false |
| `--help` | `-h` | Mostra ajuda | |
| `--version` | `-V` | Mostra versao | |

### Derivacao automatica (branch -> worktree)

O nome da worktree e derivado da branch automaticamente:

1. `/` vira `-`
2. camelCase vira kebab-case
3. Tudo minusculo

```
feature/loginForm      -> feature-login-form
bugfix/fixApiTimeout   -> bugfix-fix-api-timeout
hotfix-urgent          -> hotfix-urgent
```

## Exemplos

```bash
# Basico — cria worktree ../feature-login-form com branch feature/loginForm
wt feature/loginForm

# Nome explicito da worktree
wt feature/loginForm --name meu-fix

# Base branch diferente da atual
wt feature/loginForm --base main

# Path customizado
wt feature/loginForm --path ./branchs

# Ver o que seria feito sem executar
wt feature/loginForm --dry-run

# Sem copia de .worktreeinclude
wt feature/loginForm --no-include
```

## .worktreeinclude

Crie um arquivo `.worktreeinclude` na raiz do repositorio listando arquivos gitignored que devem ser copiados para novas worktrees. Usa a sintaxe completa do `.gitignore`, incluindo padroes de negacao.

```gitignore
# .worktreeinclude

# Copiar todos os .env, exceto o de producao
*.env
!prod.env

# Configs locais
.env.local
lib/firebase_options.dart
android/app/google-services.json
ios/Runner/GoogleService-Info.plist
.vscode/
```

Apenas arquivos que sao **gitignored e untracked** serao copiados. Arquivos tracked pelo git ja estao presentes na worktree naturalmente.

## Requisitos

- git
