class SessionEvent {
  final String type;
  final String sessionId;
  final String timestamp;
  final Map<String, dynamic> data;

  SessionEvent({
    required this.type,
    required this.sessionId,
    required this.timestamp,
    required this.data,
  });

  factory SessionEvent.fromJson(Map<String, dynamic> json) {
    return SessionEvent(
      type: json['type'] as String,
      sessionId: json['session_id'] as String,
      timestamp: json['timestamp'] as String,
      data: json['data'] as Map<String, dynamic>,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'type': type,
      'session_id': sessionId,
      'timestamp': timestamp,
      'data': data,
    };
  }
}