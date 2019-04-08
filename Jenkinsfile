pipeline {
    agent { 
        node {
            label 'bare-metal'
            
           
            
         }
    } 
    stages {
        stage('pre-cleanup') {
            steps {
                sh '''#!/bin/bash
                base_dir=`pwd`

                if [ -d $base_dir/Skyport2 ] ; then
                  cd $base_dir/Skyport2
                  rm -f docker-compose.yaml
                  ln -s Docker/Compose/skyport-awe-testing.yaml docker-compose.yaml
                fi
                
                # Clean up
                if [ -e $base_dir/skyport2.env ] ; then
                    cd $base_dir/Skyport2
                    source ./skyport2.env
                    docker-compose down
                fi
                DELETE_CONTAINERS=$(docker ps -a -f name=skyport2_ -q)
                if [ "${DELETE_CONTAINERS}_" != "_" ] ; then
                  docker rm -f ${DELETE_CONTAINERS}
                fi
                
                DELETE_CONTAINERS=$(docker ps -a -f name=compose_ -q)
                if [ "${DELETE_CONTAINERS}_" != "_" ] ; then
                  docker rm -f ${DELETE_CONTAINERS}
                fi
                
                DELETE_CONTAINERS=$(docker ps -a -f name=mgrast_cwl_submitter -q)
                if [ "${DELETE_CONTAINERS}_" != "_" ] ; then
                  docker rm -f ${DELETE_CONTAINERS}
                fi
                
                
                
                echo "Deleting live-data"

                docker run --rm --volume `pwd`:/tmp/workspace bash rm -rf /tmp/workspace/live-data
                
                docker volume prune -f
                '''
            }
        }
        stage('git-clone') {
            steps {
                sh '''#!/bin/bash
                git clone https://github.com/wgerlach/AWE.git
                git clone https://github.com/MG-RAST/Skyport2.git
                git clone https://github.com/common-workflow-language/common-workflow-language.git
                '''
            }
        }
        stage('Build') { 
            steps {
                sh '''#!/bin/bash
                set -e
                
                echo "SHELL=$SHELL"
                echo "HOSTNAME=$HOSTNAME"


                export PATH="/usr/local/bin/:$PATH"
                echo "PATH=$PATH"
                DOCKER_PATH=$(which docker)
                echo "DOCKER_PATH=${DOCKER_PATH}"

                base_dir=`pwd`

                
                cd $base_dir/Skyport2
                rm -f docker-compose.yaml
                ln -s Docker/Compose/skyport-awe-testing.yaml docker-compose.yaml
                
                
                
                # Debugging
                pwd
                ls -l
                # docker images
                docker ps
                
                
                
               
                docker ps
                set -x
                sudo ./scripts/add_etc_hosts_entry.sh

                source ./init.sh
                

                if [ ${SKYPORT_DOCKER_GATEWAY}x == x ] ; then
                  exit 1
                fi

                # Build container
                cd $base_dir/AWE
                set +x
                echo Building AWE Container
                set -x

                #git branch -v
                #git remote -v
                #git describe

                USE_CACHE="--no-cache"
                #USE_CACHE="" #speed-up for debugging purposes 

                docker build ${USE_CACHE} --pull -t mgrast/awe:test -f Dockerfile .
                docker build ${USE_CACHE} --pull -t mgrast/awe-worker:test -f Dockerfile_worker .
                docker build ${USE_CACHE} --pull -t mgrast/awe-submitter:test -f Dockerfile_submitter .
                cd $base_dir/Skyport2
                docker run --rm --volume `pwd`:/Skyport2 bash rm -rf /Skyport2/tmp
                docker build ${USE_CACHE} -t mgrast/cwl-submitter:test -f Docker/Dockerfiles/cwl-test-submitter.dockerfile .

                echo "docker builds complete"
                sleep 5

                docker run --rm mgrast/awe:test awe-server --version
                docker ps

                sleep 1

                if [ ${SKYPORT_DOCKER_GATEWAY}x == x ] ; then
                  set +e
                  exit 1
                fi

                echo "SKYPORT_DOCKER_GATEWAY: ${SKYPORT_DOCKER_GATEWAY}"
                export SKYPORT_DOCKER_GATEWAY=${SKYPORT_DOCKER_GATEWAY}
                docker-compose up -d
                '''
            }
        }
        stage('Test') { 
            steps {
                sh '''#!/bin/bash
                set -x
            
                base_dir=`pwd`
                cd $base_dir
                touch result.xml
                docker run \
                    --rm \
                    --network skyport2_default \
                    --name mgrast_cwl_submitter \
                    --volume `pwd`/result.xml:/output/result.xml \
                    mgrast/cwl-submitter:test \
                    --junit-xml=/output/result.xml \
                    --timeout=120
                '''
            }
        }
    }
    post {
        always {
            sh '''#!/bin/bash
            set -x
            # Clean up
            base_dir=`pwd`
            cd $base_dir/Skyport2
            docker-compose down
            docker run --rm --volume `pwd`/live-data:/live-data bash rm -rf /live-data/*
            docker volume prune -f
            '''
        }        
    }
}