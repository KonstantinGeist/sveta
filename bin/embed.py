from gpt4all import GPT4All, Embed4All
import sys

embedder = Embed4All()

while True:
    try:
        query = input()
        if query:
            embedding = embedder.embed(query)
            print("<begin>")
            for value in embedding:
                print(str(value), end="")
                print(" ", end="")
            print("<end>", end="")
    except Exception as e:
        print(e)