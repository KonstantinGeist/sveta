## Sveta

A fully local AI agent written in Go.
This project doesn't aim to be an extensible platform for building custom bots.
Rather, it's an opinionated amalgam of various techniques the author has learned about to make it easy to create virtual humans with 1 click.
The only extensible thing so far (see config.yaml) are the parameters which depend on your specific environment (GPU specs, etc.)
Abstractions are introduced only when actually needed.

## Dependencies

1. Compile llama.cpp (https://github.com/ggerganov/llama.cpp) with CUDA enabled and copy `main` and `llava-cli` as `llama.cpp` and `llava.cpp` into ./bin respectively
2.
- If you use Llama 2, download some of the 4-bit quantized models from https://huggingface.co/TheBloke/Xwin-MLewd-13B-v0.2-GGUF?not-for-all-audiences=true and copy into ./bin/llama2-roleplay.bin
- If you use Solar, download some of the 5-bit quantized models from https://huggingface.co/TheBloke/SOLAR-10.7B-Instruct-v1.0-uncensored-GGUF and copy into ./bin/solar-generic.bin
- If you use Llama 3 (default now), download some of the 5-bit quantized models from https://huggingface.co/bartowski/Lexi-Llama-3-8B-Uncensored-GGUF
3. Install Python 3 and Embed4All (used for embeddings for now):
   - pip install gpt4all
4. Download https://huggingface.co/mys/ggml_llava-v1.5-7b/resolve/main/ggml-model-q4_k.gguf and https://huggingface.co/mys/ggml_llava-v1.5-7b/resolve/main/mmproj-model-f16.gguf and copy into ./bin/llava.bin and ./bin/llava-proj.bin respectively.

With this set, you can run cmd/console/main.go or cmd/irc/main.go to interact with the AI agent. Tested on Nvidia RTX 3060.