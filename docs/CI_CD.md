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
          curl -L https://github.com/MustardSeedNetworks/niac-go/releases/latest/download/niac-linux-x86_64.tar.gz | tar xz
          sudo mv niac /usr/local/bin/
          sudo setcap cap_net_raw,cap_net_admin=eip /usr/local/bin/niac

      - name: Start NIAC simulation
        run: |
          NIAC_API_TOKEN=$(openssl rand -base64 32)
          export NIAC_API_TOKEN
          echo "NIAC_API_TOKEN=$NIAC_API_TOKEN" >> "$GITHUB_ENV"
          niac daemon &
          echo $! > niac.pid
          sleep 5
          CSRF_TOKEN=$(curl -sk \
            -H "Authorization: Bearer $NIAC_API_TOKEN" \
            https://localhost:8445/api/v1/csrf-token | jq -r '.token')
          curl --fail-with-body -sk -X POST \
            -H "Authorization: Bearer $NIAC_API_TOKEN" \
            -H "X-CSRF-Token: $CSRF_TOKEN" \
            -H "Content-Type: application/json" \
            -d "$(jq -n --arg configData "$(cat test-network.yaml)" \
              '{interface:"eth0", configData:$configData}')" \
            https://localhost:8445/api/v1/simulation

      - name: Run network tests
        run: |
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
    - apt-get update && apt-get install -y curl jq libpcap-dev
    - curl -L https://github.com/MustardSeedNetworks/niac-go/releases/latest/download/niac-linux-x86_64.tar.gz | tar xz
    - mv niac /usr/local/bin/
    - setcap cap_net_raw,cap_net_admin=eip /usr/local/bin/niac
  script:
    - export NIAC_API_TOKEN=$(openssl rand -base64 32)
    - niac daemon &
    - sleep 5
    - |
      CSRF_TOKEN=$(curl -sk \
        -H "Authorization: Bearer $NIAC_API_TOKEN" \
        https://localhost:8445/api/v1/csrf-token | jq -r '.token')
      export CSRF_TOKEN
      curl --fail-with-body -sk -X POST \
        -H "Authorization: Bearer $NIAC_API_TOKEN" \
        -H "X-CSRF-Token: $CSRF_TOKEN" \
        -H "Content-Type: application/json" \
        -d "$(jq -n --arg configData "$(cat test-config.yaml)" \
          '{interface:"eth0", configData:$configData}')" \
        https://localhost:8445/api/v1/simulation
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
                    curl -L https://github.com/MustardSeedNetworks/niac-go/releases/latest/download/niac-linux-x86_64.tar.gz | tar xz
                    sudo mv niac /usr/local/bin/
                    sudo setcap cap_net_raw,cap_net_admin=eip /usr/local/bin/niac
                '''
            }
        }
        stage('Run Simulation') {
            steps {
                sh '''
                    openssl rand -base64 32 > niac-token
                    NIAC_API_TOKEN=$(cat niac-token) niac daemon &
                    echo $! > niac.pid
                    sleep 5
                    CSRF_TOKEN=$(curl -sk \
                      -H "Authorization: Bearer $(cat niac-token)" \
                      https://localhost:8445/api/v1/csrf-token | jq -r '.token')
                    curl --fail-with-body -sk -X POST \
                      -H "Authorization: Bearer $(cat niac-token)" \
                      -H "X-CSRF-Token: $CSRF_TOKEN" \
                      -H "Content-Type: application/json" \
                      -d "$(jq -n --arg configData "$(cat config.yaml)" \
                        '{interface:"eth0", configData:$configData}')" \
                      https://localhost:8445/api/v1/simulation
                '''
            }
        }
        stage('Test') {
            steps {
                sh 'NIAC_API_TOKEN=$(cat niac-token) pytest tests/'
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

## Release Builds

Release builds run in GitHub Actions after release-please creates a `v*` tag.
The release workflow builds native Linux, macOS, and Windows artifacts on their
respective hosted runners and uploads the artifacts to the GitHub release.
