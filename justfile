set shell := ["bash", "-uc"]
set positional-arguments

project := 'godi'

# show this help
help:
	@just --list

# configure the dev environment
configure-dev:
  @echo "⬇️  installing tools"
  @go install gotest.tools/gotestsum@latest
  @go install golang.org/x/tools/cmd/godoc@latest
  @go install github.com/evilmartians/lefthook@latest
  @echo "🔧 configuring pre-commit hooks"
  @lefthook install
  @echo "👌 done, happy hacking!"

# run unit tests
test *ARGS:
  @gotestsum -- -v -race "$@" ./...

# generate documentation
doc:
  @echo "📄 is available here: http://localhost:6060/pkg/github.com/a-peyrard/godi/" && godoc -http=:6060
