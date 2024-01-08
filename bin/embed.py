from gpt4all import GPT4All, Embed4All
import sys
query = sys.argv[1]
embedder = Embed4All()
embedding = embedder.embed(query)
for value in embedding:
    print(str(value), end="")
    print(" ", end="")
print("\n", end="")
