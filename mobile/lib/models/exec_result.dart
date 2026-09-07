class ExecResult {
  const ExecResult({
    required this.stdout,
    required this.stderr,
    required this.exitCode,
    required this.durationMs,
    required this.truncated,
  });
  final String stdout, stderr;
  final int exitCode, durationMs;
  final bool truncated;
  factory ExecResult.fromJson(Map<String, dynamic> json) => ExecResult(
    stdout: json['stdout'] as String? ?? '',
    stderr: json['stderr'] as String? ?? '',
    exitCode: (json['exitCode'] as num?)?.toInt() ?? 0,
    durationMs: (json['durationMs'] as num?)?.toInt() ?? 0,
    truncated: json['truncated'] as bool? ?? false,
  );
}
