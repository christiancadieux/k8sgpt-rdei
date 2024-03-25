
# k8sgpt auth remove openai
# k8sgpt auth add -b openai --baseurl http://alien:11434/v1  -m "mistral" --password=$OPENAI_API_KEY

all:
	make all -f makefile.base
	cp bin/k8sgpt  ~/go/src/github.comcast.com/k8s-eng/rdei-k8sgpt
	cp bin/k8sgpt  /home/ccadie883/CC/Scripts/k8sgpt

run:
	kubectl get ns --no-headers| awk '{print $1}' | grep -v kube-system > /tmp/ns1
	for i in `cat /tmp/ns1`; do k8sgpt analyze -n $i -z; done
