DEPLOY_BRANCH ?= main

.PHONY: after_bench
after_bench:
	ANSIBLE_HOST_KEY_CHECKING=False \
		ansible-playbook -i ansible/hosts.yml ansible/playbook/after_bench.yml --verbose

.PHONY: before_bench
before_bench:
	ANSIBLE_HOST_KEY_CHECKING=False \
		ansible-playbook -i ansible/hosts.yml ansible/playbook/before_bench.yml  --extra-vars "deploy_branch=${DEPLOY_BRANCH}" --verbose

.PHONY: install_tools
install_tools:
	ANSIBLE_HOST_KEY_CHECKING=False \
		ansible-playbook -i ansible/hosts.yml ansible/playbook/install_tools.yml  --verbose

.PHONY: initialize
initialize:
	ANSIBLE_HOST_KEY_CHECKING=False \
		ansible-playbook -i ansible/hosts.yml ansible/playbook/initialize.yml  --verbose

# TODO: ベンチマーカーを実行するコマンドを記述する(Optional)
# 以下の例は、private-isuのベンチマーカーを実行するコマンドです。
# .PHONY: bench
# bench:
# 	ssh isucon@bench \
# 		/home/isucon/private_isu.git/benchmarker/bin/benchmarker -u /home/isucon/private_isu.git/benchmarker/userdata -t http://$(INSUTANCE1_IP)
