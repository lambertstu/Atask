class Project {
  final String path;
  final List<String> sessions;
  final DateTime? lastModified;

  Project({required this.path, required this.sessions, this.lastModified});

  factory Project.fromJson(Map<String, dynamic> json) {
    final sessionsData = json['sessions'];
    final lastModifiedStr = json['last_modified'] as String?;
    return Project(
      path: json['path'] as String,
      sessions: sessionsData == null 
          ? [] 
          : (sessionsData as List<dynamic>).map((e) => e.toString()).toList(),
      lastModified: lastModifiedStr != null 
          ? DateTime.parse(lastModifiedStr) 
          : null,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'path': path, 
      'sessions': sessions,
      'last_modified': lastModified?.toIso8601String(),
    };
  }

  String get name {
    final parts = path.split('/');
    return parts.isNotEmpty ? parts.last : path;
  }
}