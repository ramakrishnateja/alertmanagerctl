BIN_DIR=bin
APP_NAME=alertmanagerctl

ifeq ($(git_branch), master)
	vsuffix=
else
	vsuffix=edge+$(version_date)
endif


ifeq ($(go_os), windows)
	app_file_name=$(APP_NAME)_$(go_os)_$(go_arch).exe
else
	app_file_name=$(APP_NAME)_$(go_os)_$(go_arch)
endif

init:
	@mkdir -p $(BIN_DIR)

build: init
	@echo "Building ..."
	@go build -ldflags "-X 'main.VersionSuffix=$(vsuffix)' -X 'main.VCSBranch=$(git_branch)' -X 'main.BuildTime=$(build_time)' -X 'main.VCSCommitID=$(git_commit_id)' -X 'main.BuiltBy=$(built_by)' -X 'main.BuildHost=$(build_host)' -X 'main.GOVersion=$(go_version)'" -o $(BIN_DIR)/$(app_file_name)
	@echo "Binary $(app_file_name) saved in $(BIN_DIR)"

test:
	@go test -v ./...