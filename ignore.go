package outline

// defaultIgnore is shared by Pack and Build.
var defaultIgnore = []byte(`
.git/
.hg/
.svn/
.jj/

node_modules/
bower_components/
vendor/
.bundle/
.yarn/
Pods/
Carthage/
.gradle/
target/
_build/
deps/
.venv/
venv/
__pycache__/
.mypy_cache/
.pytest_cache/
.ruff_cache/
.tox/
.terraform/

dist/
build/
out/
obj/
coverage/
.nyc_output/
.next/
.nuxt/
.svelte-kit/
.cache/
.parcel-cache/
.turbo/

*.min.js
*.min.css
*.map
*.lock
*.log
*.tmp
*.swp
*.bak
*.pyc
*.pyo
*.class
*.o
*.so
*.dylib
*.dll
*.exe
*.wasm

package-lock.json
yarn.lock
pnpm-lock.yaml
bun.lockb
Gemfile.lock
Cargo.lock
poetry.lock
composer.lock
go.sum
mix.lock

.DS_Store
Thumbs.db
.idea/
.vscode/
*.iml
`)

// packIgnore removes executable entrypoints from packed context. Build keeps
// them so source graphs include repository commands.
var packIgnore = []byte("bin/\n")
