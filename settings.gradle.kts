rootProject.name = "KeyValueStorage"

val projects = HashSet<String>()
val projectMarkers = listOf("go.mod", "build.gradle.kts")

rootDir.walkTopDown().forEach { file ->
    val path = file.toString()
    val isMarkerFile = projectMarkers.any { path.endsWith(it) }
    if (isMarkerFile) {
        val projectDir = file.parentFile
        projects.add(toProjectPath(projectDir, rootDir))
    }
}

include(*projects.toTypedArray())

fun toProjectPath(projectDir: File, rootDir: File): String {
    return projectDir.relativeTo(rootDir).toString().replace(File.separator, ":").prependIndent(":")
}
