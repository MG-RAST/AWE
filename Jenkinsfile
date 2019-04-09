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
                set -x
                base_dir=`pwd`

                
                # Clean up
                docker-compose down
                
                DELETE_CONTAINERS=$(docker ps -a -f name=skyport2_ -q)
                if [ "${DELETE_CONTAINERS}_" != "_" ] ; then
                  docker rm -f ${DELETE_CONTAINERS}
                fi
                
                DELETE_CONTAINERS=$(docker ps -a -f name=compose_ -q)
                if [ "${DELETE_CONTAINERS}_" != "_" ] ; then
                  docker rm -f ${DELETE_CONTAINERS}
                fi
                
                DELETE_CONTAINERS=$(docker ps -a -f name=awe-submitter-testing -q)
                if [ "${DELETE_CONTAINERS}_" != "_" ] ; then
                  docker rm -f ${DELETE_CONTAINERS}
                fi
                
                
                
                echo "Deleting all data in "`pwd`

                docker run --rm --volume `pwd`:/tmp/workspace bash rm -rf /tmp/workspace/live-data
                
                docker volume prune -f
                '''
            }
        }
        stage('git-clone') {
            steps {
                sh '''#!/bin/bash
                set -x 
                
                echo `pwd`
                
                rm -rf AWE
                #git clone https://github.com/wgerlach/AWE.git         # should not be needed
                
                docker run --rm --volume `pwd`:/tmp/workspace bash rm -rf /tmp/workspace/Skyport2
                git clone https://github.com/MG-RAST/Skyport2.git
                
                rm -rf common-workflow-language
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

                echo "base_dir: ${base_dir}"
                
                cd $base_dir/Skyport2
               
                
                
                
                # Debugging
                pwd
                ls -l
                # docker images
                docker ps
                
                
                
               
                docker ps
                set -x
                sudo ./scripts/add_etc_hosts_entry.sh

                ./init.sh
                

               

                # Build container
                cd ${base_dir}
                set +x
                echo Building AWE Container
                set -x

               

                USE_CACHE="--no-cache"
                USE_CACHE="" #speed-up for debugging purposes 

                docker build ${USE_CACHE} --pull -t mgrast/awe-server -f Dockerfile .
                docker build ${USE_CACHE} --pull -t mgrast/awe-worker -f Dockerfile_worker .
                docker build ${USE_CACHE} --pull -t mgrast/awe-submitter -f Dockerfile_submitter .
                
                cd ${base_dir}/testing/
                docker build ${USE_CACHE} -t mgrast/awe-submitter-testing  .
                
                cd ${base_dir}/Skyport2
                docker run --rm --volume `pwd`:/Skyport2 bash rm -rf /Skyport2/tmp
                

                echo "docker builds complete"
                sleep 5

                docker run --rm mgrast/awe-server awe-server --version
                docker ps

                sleep 1

                

                
                export TAG=latest
                docker-compose run -d awe-server
                
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
                    --env "SHOCK_SERVER=http://shock:7445" \
                    --env "AWE_SERVER=http://awe-server:8001" \
                    --network skyport2_default \
                    --name awe-submitter-testing \
                    --volume `pwd`/result.xml:/output/result.xml \
                    mgrast/awe-submitter-testing \
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