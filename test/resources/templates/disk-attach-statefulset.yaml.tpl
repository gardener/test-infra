---
apiVersion: v1
kind: Service
metadata:
  name: {{ .Name }}
  labels:
    app: {{ .Name }}
spec:
  ports:
  - port: 80
    name: web
  clusterIP: None
  selector:
    app: {{ .Name }}
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ .Name }}
spec:
  serviceName: "{{ .Name }}"
  replicas: 1
  selector:
    matchLabels:
      app: {{ .Name }}
  template:
    metadata:
      labels:
        app: {{ .Name }}
    spec:
      containers:
      - name: nginx
        image: k8s.gcr.io/nginx-slim:0.8
        ports:
        - containerPort: 80
          name: web
        volumeMounts:
        - name: www
          mountPath: /usr/share/nginx/html
  volumeClaimTemplates:
  - metadata:
      name: www
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: {{ .PVCStorage }}

