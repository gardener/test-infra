#!/usr/bin/env python3

"""
Test application to validate networking between cluster nodes and pods is
working. Control plane test will only work on a seed cluster but apart from
that test will run on any cluster.
"""

# Notes:
# - python kubernetes client fails on connect_get_namespaced_pod_exec ("Upgrade request required")
# - other calls should be migrated to the python k8s client


import subprocess
import sys
import re
import json
import argparse
import os
import time
from kubernetes import client, config
from kubernetes.client.rest import ApiException
from kubernetes.stream import stream

#----------------------------------------------------------------------------
exec_snippet="kubectl exec -it -n {} {} -c {} -- {}"
cp_snippet="kubectl cp {} {}/{}:{} -c {}"

get_control_plane_pods="kubectl get pods -n %s -o json | jq '[.items[]|{\"name\": .metadata.name,\"hostIP\": .status.hostIP,\"podIP\": .status.podIP}]'"

PINGMANY_FILE = os.path.join(os.path.dirname(__file__), "pingmany")
DAEMONSET_FILE = os.path.join(os.path.dirname(__file__), "network-test-daemonset.yaml")

POD_RUNNING_STATUS = "Running"
#----------------------------------------------------------------------------


class Node:
    def __init__(self, ip, name):
        self.name=name
        self.ip=ip
        self.can_reach = []
        self.can_not_reach = []
        self.error = None

    def __str__(this):
        can_reach_nodes = " ".join(this.can_reach)
        return (this.name + " " + this.ip + " " + this.calico_pod_name + "\n" +
                "Can reach nodes: " + can_reach_nodes)

    def read_nodes():
        v1 = client.CoreV1Api()
        res = v1.list_node()
        nodes = []
        for i in res.items:
            n = Node(ip=i.status.addresses[0].address, name=i.metadata.name)
            nodes.append(n)
        return nodes


class Pod:
    def read_pods(field_selector=None, label_selector=None):
        v1 = client.CoreV1Api()
        params = {}
        if field_selector is not None:
            params["field_selector"] = field_selector
        if label_selector is not None:
            params["label_selector"] = label_selector
        res = v1.list_pod_for_all_namespaces(**params)
        pods = []
        for i in res.items:
            p = Pod()
            p.name = i.metadata.name
            p.namespace = i.metadata.namespace
            p.hostIP = i.status.host_ip
            p.podIP  = i.status.pod_ip
            p.status = i.status.phase
            if hasattr(i.status, "container_statuses") and i.status.container_statuses is not None:
                p.containers = []
                for cs in i.status.container_statuses:
                    if hasattr(cs, "container_id") and hasattr(cs, "image"):
                        c = Container()
                        c.containerID = cs.container_id
                        c.image = cs.image
                        p.containers.append(c)
            pods.append(p)
        return pods


class Namespace:
    def read_ns():
        v1 = client.CoreV1Api()
        res = v1.list_namespace()
        ns = []
        for i in res.items:
            ns.append(Namespace(name=i.metadata.name))
        return ns

    def __init__(this, name):
        this.name = name

    def __str__(this):
        return this.name


class Container:
    pass


class ControlPlane:
    pass


def get_cluster_nodes():
    nodes = Node.read_nodes()
    nodeMap = {}
    for i in nodes:
        nodeMap[str(i.ip)] = i

    c_pods = Pod.read_pods(field_selector="metadata.namespace=kube-system",label_selector="k8s-app=calico-node")
    for p in c_pods:
        n = nodeMap[p.hostIP]
        n.calico_pod_name = p.name
    return nodeMap


def copy_script_to_pod(pod):
    cp_cmd = cp_snippet.format(PINGMANY_FILE, "kube-system", pod, "pingmany", "calico-node")
    res = subprocess.run(cp_cmd, shell=True, capture_output=True, check=True, encoding="utf-8")


def rm_pingmany(ns, pod, container):
    cmd = exec_snippet.format(ns, pod, container, "rm /pingmany")
    # ignore errors here
    subprocess.run(cmd, shell=True, capture_output=True, check=False, encoding="utf-8")


def parse_ping_out(node, output):
    pt = re.compile('([\d]+.[\d]+.[\d]+.[\d]+) ping statistics')
    lines = output.split("\n")
    while len(lines) > 0:
        s = lines.pop(0)
        match = re.search(pt, s)
        if match is not None:
            loss_line = lines.pop(0)
            if re.search("1 received", loss_line):
                node.can_reach.append(match.group(1))
            else:
                node.can_not_reach.append(match.group(1))


def run_ping_test(nodes):
    all_nodes = nodes.keys()
    for k in nodes:
        n = nodes[k]
        print("Running test on node " + n.ip)
        other_nodes = list(filter(lambda x: x != n.ip, all_nodes))
        try:
            copy_script_to_pod(n.calico_pod_name)
            cmd = exec_snippet.format("kube-system", n.calico_pod_name, "calico-node", "/pingmany " + " ".join(other_nodes))
            res = subprocess.run(cmd, shell=True, capture_output=True, check=True, encoding="utf-8")
            if res.returncode != 0:
                n.error = "ping command returned with code {}. Output is {}".format(str(res.returncode), res.stdout)
            else:
                ping_out = res.stdout
                parse_ping_out(n, ping_out)
        except subprocess.CalledProcessError as e:
            n.error = "{} failed with {}: {} {}".format(e.cmd, str(e.returncode), e.stdout, e.stderr)
        finally:
            rm_pingmany("kube-system", n.calico_pod_name, "calico-node")


def print_statistics(nodes_map):
    nodes = nodes_map.values()
    all_ok = list(filter(lambda x: (hasattr(x, "error") and x.error is not None) or len(x.can_not_reach) > 0, nodes))
    if len(all_ok) == 0:
        print("All tests successful")
    else:
        for node in nodes:
            if node.error is not None:
                print("Error running test on node " + node.ip)
                print(node.error)
            else:
                if len(node.can_not_reach) > 0:
                    print("Node {} can not reach nodes {}".format(node.ip, " ".join(node.can_not_reach)))


def examine_shoot_control_plane(ns):
    print("Examine control plane " + ns)
    nodes = subprocess.run(get_control_plane_pods % ns, shell=True, capture_output=True, check=True, encoding="utf-8")
    control_plane = json.loads(nodes.stdout)
    kube_apiserver_name = None
    etcd_ip = None
    for cp in control_plane:
        kube_apiserver_name = cp["name"] if cp["name"].startswith("kube-apiserver") else kube_apiserver_name
        etcd_ip = cp["podIP"] if cp["name"].startswith("etcd-main-0") else etcd_ip

    if kube_apiserver_name is not None and etcd_ip is not None:
        ping_exec = exec_snippet.format(ns, kube_apiserver_name, "kube-apiserver", "ping -W 2 -c 1 " + etcd_ip)
        print(ping_exec)
        r = subprocess.run(ping_exec, shell=True, capture_output=True, encoding="utf-8")
        print(r.stdout)


def get_cp_pods():
    all_pods = Pod.read_pods()
    return list(filter(lambda x: x.namespace != "kube-system", all_pods))


def get_container_id(pod, id):
    for c in pod.containers:
        if c.image.find(id) != -1:
            if c.containerID.startswith("docker://"):
                return c.containerID[9:]
    return None


def ping_etcd_from_apiserver(api_server, root_pod_map, etcd_pod):
    if api_server.status != POD_RUNNING_STATUS:
        print("{}/{} is not running (status: {}). Ignoring".format(api_server.namespace, api_server.name, api_server.status))
        return True
    print("kube-apiserver {}/{} connectivity test with {}/{}".format(api_server.namespace, api_server.name, etcd_pod.namespace, etcd_pod.name))
    containerID = get_container_id(api_server, "k8s.gcr.io/hyperkube")
    if containerID is None:
        print("fail: {}/{} pod has no hyperkube container".format(api_server.namespace, api_server.name))
        # fail the test
        return False

    root_pod = root_pod_map[api_server.hostIP]

    cmd = "kubectl exec -it {} -- chroot /root docker inspect {}".format(root_pod.name, containerID)
    print("Running " + cmd)
    inspect = subprocess.run(cmd, shell=True, capture_output=True, encoding="utf-8")
    if inspect.returncode != 0:
        print("Unable to run test from pod " + api_server.name + ": " + inspect.stderr)
        return False
    inspect_json = json.loads(inspect.stdout)
    api_server_pid = inspect_json[0]["State"]["Pid"]

    cmd = "kubectl exec -it {} -- nsenter -n/proc/{}/ns/net  -- nc -vz -w 2 {} {}".format(root_pod.name, api_server_pid, etcd_pod.podIP, 2379)
    print("Running " + cmd)
    nc = subprocess.run(cmd, shell=True, capture_output=True, encoding="utf-8")
    if nc.returncode != 0:
        print("failed: " + nc.stdout + nc.stderr)
        return False
    else:
        print("success")
        return True


def check_etcd_from_apiservers(node_map, pods, seeds):

    selected_seeds = set()
    if seeds is not None:
        for i in seeds:
            selected_seeds.add(i)
    etcds = filter(lambda x: x.name == "etcd-main-0", pods)
    ntpods = filter(lambda x: x.name.startswith("network-test-"), pods)
    api_servers = filter(lambda x: x.name.startswith("kube-apiserver"), pods)
    ntmap = {}
    for n in ntpods:
        ntmap[n.podIP] = n

    shoot_etcd_map = {}
    for etcd in etcds:
        shoot_etcd_map[etcd.namespace] = etcd

    test_success = True
    for aserver in api_servers:
        ns = aserver.namespace
        if len(selected_seeds) > 0 and ns not in selected_seeds:
            continue
        target_etcd = shoot_etcd_map[ns]
        aserver.target_etcd = target_etcd
        result = ping_etcd_from_apiserver(aserver, ntmap, target_etcd)
        if result == False:
            test_success = False
    return test_success


def get_control_plane_namespaces():
    ns = Namespace.read_ns()
    j = list(filter(lambda x: x.name.startswith("shoot--"), ns))
    return j


def is_deamon_set_running():
    num_nodes = len(Node.read_nodes())
    retries = 0
    pods_running = 0
    while pods_running != num_nodes and retries < 20:
        time.sleep(2)
        retries = retries + 1
        pods_running = 0
        pods = Pod.read_pods(label_selector="k8s-app=network-test")
        for i in pods:
            print("{} {}".format(i.name, i.status))
            if i.status == POD_RUNNING_STATUS:
                pods_running = pods_running + 1
    if pods_running == num_nodes:
        return True
    else:
        return False


def deploy_root_daemonset():
    if not os.path.isfile(DAEMONSET_FILE):
        print("Daemonset {} is missing".format(DAEMONSET_FILE))
        sys.exit(1)
    print("Deploying test daemonset.")
    cmd = "kubectl apply -f " + DAEMONSET_FILE
    res = subprocess.run(cmd, shell=True, capture_output=True, check=True, encoding="utf-8")
    if res.returncode != 0:
        print("Error deploying {}: {} {}".format(DAEMONSET_FILE, res.stdout, res.stderr))
        sys.exit(1)

    # wait until running
    print("Waiting for network-test daemon set to come up")
    up = is_deamon_set_running()
    if not up:
        print("network-test daemonset did not start correctly")
        sys.exit(1)


def undeploy_root_daemonset():
    cmd = "kubectl delete daemonset network-test"
    # nothing we can do if this fails
    res = subprocess.run(cmd, shell=True, capture_output=True, check=False, encoding="utf-8")

#----------------------------------------------------------------------------

def check_env():

    if not "KUBECONFIG" in os.environ:
        sys.stderr.write("No KUBECONFIG set.\n")
        sys.exit(1)
    if not os.path.isfile(os.environ["KUBECONFIG"]):
        sys.stderr.write("Referenced KUBECONFIG file {} does not exist.\n".format(os.environ["KUBECONFIG"]))
        sys.exit(1)

def init():
    check_env()
    config.load_kube_config()


def main():

    init()
    parser = argparse.ArgumentParser(description="Seed cluster connectivity test.")
    parser.add_argument("--nodes", action="store_true", help="node connectivity test")
    parser.add_argument("--control-planes", action="store_true", help="control plane components connectivity test")
    parser.add_argument("--seed", action="append", help="seed cluster namespace (seed--<project>-->name> (use multiple times for multiple control planes)")
    args = parser.parse_args()
    if not (args.nodes or args.control_planes):
        parser.print_help()
        sys.exit(1)

    test_success = True
    try:
        node_map = get_cluster_nodes()
        if args.nodes:
            # there is no point in trying to do this if this script is missing
            if not os.path.isfile(PINGMANY_FILE):
                print("Missing script \"pingmany\”")
                sys.exit(1)
            print("Running node connectivity test.")
            run_ping_test(node_map)
            print_statistics(node_map)
            for i in node_map.values():
                if i.error is not None:
                    test_success = False
        if args.control_planes:
            ns = get_control_plane_namespaces()
            if len(ns) == 0:
                print("The are no control planes in this cluster.")
                sys.exit(1)
            selected_shoots = set()
            if args.seed is not None:
                # only test names seeds but make sure they exist
                ns_set = set()
                for i in ns:
                    ns_set.add(i.name)
                for i in args.seed:
                    if not i in ns_set:
                        print("Control plane for cluster {} not in seed.".format(i))
                        sys.exit(1)
            deploy_root_daemonset()
            pods = get_cp_pods()
            success = check_etcd_from_apiservers(node_map, pods, args.seed)
            if not success:
                test_success = False
    except subprocess.CalledProcessError as e:
        print(e)
        print(e.stdout)
        print(e.stderr)
        test_success = False
    finally:
        undeploy_root_daemonset()

    if test_success is True:
        sys.exit(0)
    else:
        print("At least one test failed.")
        sys.exit(1)

if __name__ == "__main__":
    main()