SDK_VERSION=14.3.0
#Linux
.PHONY: setup
setup:
	spatial package get worker_sdk c-dynamic-x86_64-gcc510-linux ${SDK_VERSION} ./c_sdk --unzip
	spatial package get worker_sdk c_headers ${SDK_VERSION} ./c_sdk --unzip

#Windows
.PHONY: setup_win
setup_win:
	spatial package get worker_sdk c-dynamic-x86_64-vc140_md-win32 ${SDK_VERSION} ./c_sdk --unzip
	spatial package get worker_sdk c_headers ${SDK_VERSION} ./c_sdk --unzip

