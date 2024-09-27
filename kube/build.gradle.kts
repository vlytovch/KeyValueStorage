task<Exec>("deploy") {
    commandLine("kubectl", "apply", "-f", "$projectDir/templates/storage-deployment.yaml")
}

task<Exec>("undeploy") {
    commandLine("kubectl", "delete", "-f", "$projectDir/templates/storage-deployment.yaml")
}
