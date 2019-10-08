pipeline {
    options {
        ansiColor('xterm')
    }
    agent { 
        node {
            label 'bare-metal||awe'
            
           
            
         }
    } 
    stages {
        stage('pre-cleanup') {
            steps {
                sh '''#!/bin/bash
                set -x
                base_dir=`pwd`

                
                # Clean up
                cd ${base_dir}/test
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

                docker run --rm  -v ${base_dir}:/basedir bash rm -rf /basedir/tmp /basedir/data


                docker volume prune -f
                echo "stage pre-cleanup done"
                '''
            }
        }
        stage('git-clone') {
            steps {
                sh '''#!/bin/bash
                set -x 
                
                echo `pwd`
                
                
                # this is needed for AWE compile scripts to determine the version number
                git fetch --tags
                
                
                #git pull
                
                
                docker run --rm --volume `pwd`:/tmp/workspace bash rm -rf /tmp/workspace/Skyport2
                git clone https://github.com/MG-RAST/Skyport2.git
                
                rm -rf common-workflow-language
                git clone https://github.com/common-workflow-language/common-workflow-language.git
                
                echo "stage git-clone done"
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
            
                cd ${base_dir}/Skyport2
           
                
            
                # Debugging
                pwd
                ls -l
                # docker images
                docker ps
            
            
            
           
                docker ps
                set -x
                
           

                # Build container
                cd ${base_dir}
                set +x
                echo Building AWE Container
                set -x

           

                #USE_CACHE="--no-cache"
                USE_CACHE="" #speed-up for debugging purposes 
                set +e
                #docker rmi mgrast/awe-server mgrast/awe-worker mgrast/awe-submitter
                set -e
                sleep 2
                
                docker pull mgrast/shock
                docker build ${USE_CACHE} --pull -t mgrast/awe-server .
                docker build ${USE_CACHE} --pull -t mgrast/awe-worker -f Dockerfile_worker .
                docker build ${USE_CACHE} --pull -t mgrast/awe-submitter -f Dockerfile_submitter .
            
                cd ${base_dir}/test/
                docker build ${USE_CACHE} -t mgrast/awe-submitter-testing  .
            
                #cd ${base_dir}/Skyport2
                #docker run --rm --volume `pwd`:/Skyport2 bash rm -rf /Skyport2/tmp
            

                echo "docker builds complete"
                sleep 5

                docker run --rm mgrast/awe-server awe-server --version
                docker ps

                sleep 1
                
                
                export SKYPORT_TMPDIR=${base_dir}/tmp
                export DATADIR=${base_dir}/data
                mkdir -p ${SKYPORT_TMPDIR}
                mkdir -p ${DATADIR}
                
                
                cd ${base_dir}/Skyport2/scripts/
                source ./get_docker_binary.sh 

                cd ${base_dir}
               
                
                set +e
                rm -f ./result.xml
                set -e

                cd ${base_dir}/test
               

                docker-compose up -d
                set +e
                sleep 3
                docker ps
                docker network ls
                sleep 1
                echo "services started"
                '''
        
            }
        }
        stage('Test') { 
            steps {
                sh '''#!/bin/bash
                set -x
            
                base_dir=`pwd`
                cd $base_dir
                
                
                # network name sometimes has a minus, sometimes not, depends on docker-compose version
                NETWORK_NAME=$(docker network ls  | grep -o "awe-\\?test_default")
                echo "NETWORK_NAME: ${NETWORK_NAME}"
                set +e
               
                
                touch result.xml
                docker run \
                    --rm \
                    --env "SHOCK_SERVER=http://shock:7445" \
                    --env "AWE_SERVER=http://awe-server:80" \
                    --network ${NETWORK_NAME} \
                    --name awe-submitter-testing \
                    --volume `pwd`/result.xml:/output/result.xml \
                    mgrast/awe-submitter-testing \
                    --junit-xml=/output/result.xml \
                    --timeout=120
                    
                echo "stage Test done"
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
            cd $base_dir/test
            docker-compose down
            #docker run --rm --volume `pwd`/live-data:/live-data bash rm -rf /live-data/*
            docker volume prune -f
            echo "stage clean-up done"
            '''
            junit "result.xml"
        }        
    }
}