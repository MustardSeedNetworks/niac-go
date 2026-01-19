# CI/CD Integration Examples

## GitHub Actions

```.github/workflows/test-with-niac.yml
name: Network Tests

on: [push, pull_request]

jobs:
  network-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install NIAC-Go
        run: |
          curl -L https://github.com/krisarmstrong/niac-go/releases/latest/download/niac-linux-x86_64.tar.gz | tar xz
          sudo mv niac /usr/local/bin/
          sudo setcap cap_net_raw,cap_net_admin=eip /usr/local/bin/niac

      - name: Start NIAC simulation
        run: |
          sudo niac run test-network.yaml --api :8080 &
          echo $! > niac.pid
          sleep 5

      - name: Run network tests
        run: |
          export NIAC_API_TOKEN=$(openssl rand -base64 32)
          pytest tests/network_tests.py

      - name: Stop NIAC
        if: always()
        run: |
          sudo kill $(cat niac.pid) || true
```

## GitLab CI

```.gitlab-ci.yml
network_tests:
  image: ubuntu:22.04
  before_script:
    - apt-get update && apt-get install -y curl libpcap-dev
    - curl -L https://github.com/krisarmstrong/niac-go/releases/latest/download/niac-linux-x86_64.tar.gz | tar xz
    - mv niac /usr/local/bin/
    - setcap cap_net_raw,cap_net_admin=eip /usr/local/bin/niac
  script:
    - niac run test-config.yaml --api :8080 &
    - sleep 5
    - python3 run_tests.py
    - kill $(pgrep niac)
```

## Jenkins Pipeline

```groovy
pipeline {
    agent any
    stages {
        stage('Setup NIAC') {
            steps {
                sh '''
                    curl -L https://github.com/krisarmstrong/niac-go/releases/latest/download/niac-linux-x86_64.tar.gz | tar xz
                    sudo mv niac /usr/local/bin/
                '''
            }
        }
        stage('Run Simulation') {
            steps {
                sh '''
                    sudo niac run config.yaml --api :8080 &
                    echo $! > niac.pid
                    sleep 5
                '''
            }
        }
        stage('Test') {
            steps {
                sh 'pytest tests/'
            }
        }
    }
    post {
        always {
            sh 'sudo kill $(cat niac.pid) || true'
        }
    }
}
```

## Docker-based CI

```yaml
test:
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker build -t niac-test .
    - docker run --rm --privileged --network host niac-test niac run /config/test.yaml --api :8080 &
    - sleep 5
    - docker run --rm --network host test-runner pytest
```
