## Sveta

A fully local AI agent written in Go.

## Dependencies

1. Compile llama.cpp (https://github.com/ggerganov/llama.cpp) with CUDA enabled and copy `main` as `llama.cpp` into ./bin
2. Download some of the 4-bit quantized models from https://huggingface.co/TheBloke/Xwin-MLewd-13B-v0.2-GGUF?not-for-all-audiences=true and copy into ./bin/llama2.bin
3. Install Python 3 and Embed4All (used for embeddings for now).

With this set, you can run cmd/console/main.go or cmd/irc/main.go to interact with the AI agent. Tested on Nvidia RTX 3060.