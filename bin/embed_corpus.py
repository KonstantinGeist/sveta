from gpt4all import GPT4All, Embed4All
from text_chunker import TextChunker

embedder = Embed4All(verbose=False)
chunker = TextChunker(maxlen=500)

output_file = open("chunks.bin", "w")

with open("corpus.txt") as file:
    allText = file.read()
    allText = allText.replace("\n", " ")
    for line in chunker.chunk(allText):
        if line == "":
            continue
        output_file.write(line + "\n")
        embedding = embedder.embed(line)
        for value in embedding:
            output_file.write(str(value))
            output_file.write(" ")
        output_file.write("\n")