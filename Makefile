.PHONY: wasm serve

wasm:
	cp $$(tinygo env TINYGOROOT)/targets/wasm_exec.js .
	tinygo build -o main.wasm -target wasm main_wasm.go

serve:
	python3 server.py
