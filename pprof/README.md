# Profiling Operator Controller

The TCP socket to be used to serve pprof is bound to the localhost / 127.0.0.1 IP address to protect if from being accessed externally.

```Golang
    mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme.Scheme,
		Metrics:                server.Options{BindAddress: metricsAddr},
		PprofBindAddress:       "127.0.0.1:8989",
        ...
		},
```

However, should you work to access it during troubleshooting, you can port forward the socket to you local development system as shown below:

```Bash
operator-controller/ (profiling✗) $ kubectl -n olmv1-system get po
NAME                                                     READY   STATUS    RESTARTS        AGE
catalogd-controller-manager-75457dc88c-jsps2             2/2     Running   1 (5h52m ago)   5h57m
operator-controller-controller-manager-f95dc4674-4kzws   2/2     Running   0               5h57m
operator-controller/ (profiling✗) $ 
```

Once you identify the name of the name of the pods, get the name of the operator controller pod and port forward to port `8989` on the pod. 

```Bash
operator-controller/ (profiling✗) $ kubectl -n olmv1-system port-forward operator-controller-controller-manager-f95dc4674-4kzws 8989
Forwarding from 127.0.0.1:8989 -> 8989
Forwarding from [::1]:8989 -> 8989
Handling connection for 8989
Handling connection for 8989
Handling connection for 8989
```

In separate terminals, start collecting pprof data cpu, memory heap and trace profiles for the application.


```Bash
operator-controller/ (profiling) $ curl -o pprof/cpu_profile.pprof http://localhost:8989/debug/pprof/profile?seconds=600
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100  115k    0  115k    0     0    197      0 --:--:--  0:10:00 --:--:-- 32064s
operator-controller/ (profiling✗) $
```

```Bash
operator-controller/ (profiling) $ curl -o pprof/mem_heap_profile.pprof http://localhost:8989/debug/pprof/heap?seconds=600
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100 48071    0 48071    0     0     80      0 --:--:--  0:10:00 --:--:-- 14961
operator-controller/ (profiling✗) $
```

```Bash
operator-controller/ (profiling) $ curl -o pprof/trace_profile.pprof http://localhost:8989/debug/pprof/trace?seconds=600
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100 13.0M    0 13.0M    0     0  22867      0 --:--:--  0:10:00 --:--:-- 11937
operator-controller/ (profiling✗) $
```

With the above terminals and port-forward running, perform any user actions such as installing a cluster extension.

```Bash
operator-controller/ (profiling✗) $ kubectl apply -f config/samples/olm_v1alpha1_clusterextension.yaml
clusterextension.olm.operatorframework.io/clusterextension-sample unchanged
operator-controller/ (profiling✗)
```