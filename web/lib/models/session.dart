import 'message.dart';

enum SessionState { pending, planning, processing, blocked, completed }

enum SessionMode { plan, build }

class Session {
  final String id;
  final String projectPath;
  final String model;
  final SessionState state;
  final SessionMode mode;
  final String createdAt;
  final String input;
  final List<ChatMessage> messages;
  final String? blockedOn;
  final String? blockedTool;
  final Map<String, dynamic>? blockedArgs;

  Session({
    required this.id,
    required this.projectPath,
    required this.model,
    required this.state,
    required this.mode,
    required this.createdAt,
    required this.input,
    required this.messages,
    this.blockedOn,
    this.blockedTool,
    this.blockedArgs,
  });

  factory Session.fromJson(Map<String, dynamic> json) {
    return Session(
      id: json['id'] as String,
      projectPath: json['project_path'] as String,
      model: json['model'] as String,
      state: SessionState.values.firstWhere(
        (e) => e.name == json['state'],
        orElse: () => SessionState.pending,
      ),
      mode: SessionMode.values.firstWhere(
        (e) => e.name == json['mode'],
        orElse: () => SessionMode.build,
      ),
      createdAt: json['created_at'] as String,
      input: json['input'] as String? ?? '',
      messages: _parseMessages(json['messages']),
      blockedOn: json['blocked_on'] as String?,
      blockedTool: json['blocked_tool'] as String?,
      blockedArgs: json['blocked_args'] as Map<String, dynamic>?,
    );
  }

  static List<ChatMessage> _parseMessages(dynamic messagesData) {
    if (messagesData == null) return [];
    return (messagesData as List<dynamic>)
        .map((e) => ChatMessage.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'project_path': projectPath,
      'model': model,
      'state': state.name,
      'mode': mode.name,
      'created_at': createdAt,
      'input': input,
      'messages': messages.map((e) => e.toJson()).toList(),
      'blocked_on': blockedOn,
      'blocked_tool': blockedTool,
      'blocked_args': blockedArgs,
    };
  }

  String get projectName {
    final parts = projectPath.split('/');
    return parts.isNotEmpty ? parts.last : projectPath;
  }
}