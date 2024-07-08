#!/usr/bin/env python3
import subprocess
import json
import time
import sys

def catalogd_metrics(pod_name: str) -> str:
    result = subprocess.run(["kubectl","get","--raw", "/apis/metrics.k8s.io/v1beta1/namespaces/olmv1-system/pods"], stdout=subprocess.PIPE)
    data = json.loads(result.stdout)

    for item in data['items']:
        if item['metadata']['name'] == pod_name:
            for container in item['containers']:
                if container['name'] == 'manager':
                    return '{},{},{},{}\n'.format(
                            pod_name, 
                            'manager', 
                            container['usage']['cpu'], 
                            container['usage']['memory'])

def main(pod_name: str) -> None:
    count = 0
    metrics_filename = '{}_metrics.csv'.format(pod_name)
    with open(metrics_filename, 'w') as f:
        while True:
            metrics = catalogd_metrics(pod_name)
            f.write(metrics)
            time.sleep(5)
            count += 1
            if count >= 24:
                break

if __name__ == '__main__':
    pod_name = sys.argv[1]
    main(pod_name)
