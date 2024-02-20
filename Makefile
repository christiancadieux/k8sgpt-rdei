
# k8sgpt auth remove openai
# k8sgpt auth add -b openai --baseurl http://alien:11434/v1  -m "mistral" --password=$OPENAI_API_KEY

all:
	make all -f makefile.base
	cp bin/k8sgpt  ~/go/src/github.comcast.com/k8s-eng/rdei-k8sgpt
	cp bin/k8sgpt  /home/ccadie883/CC/Scripts/k8sgpt

