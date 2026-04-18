import 'package:flutter/material.dart';
import '../models/session.dart';

class ApprovalDialog extends StatelessWidget {
  final Session session;
  final Function(bool) onDecision;
  
  const ApprovalDialog({
    super.key,
    required this.session,
    required this.onDecision,
  });
  
  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Tool Approval Required'),
      content: SingleChildScrollView(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Text('Session: ${session.id.substring(0, 8)}'),
            const SizedBox(height: 8),
            Text('Blocked Tool: ${session.blockedTool ?? "Unknown"}'),
            const SizedBox(height: 8),
            if (session.blockedArgs != null) ...[
              const Text('Arguments:', style: TextStyle(fontWeight: FontWeight.bold)),
              const SizedBox(height: 4),
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: Colors.grey[200],
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Text(
                  _formatArgs(session.blockedArgs!),
                  style: const TextStyle(fontSize: 12),
                ),
              ),
            ],
            const SizedBox(height: 16),
            const Text(
              'Do you approve this tool execution?',
              style: TextStyle(fontWeight: FontWeight.bold),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () {
            Navigator.pop(context);
            onDecision(false);
          },
          child: const Text('Reject', style: TextStyle(color: Colors.red)),
        ),
        ElevatedButton(
          onPressed: () {
            Navigator.pop(context);
            onDecision(true);
          },
          style: ElevatedButton.styleFrom(backgroundColor: Colors.green),
          child: const Text('Approve'),
        ),
      ],
    );
  }
  
  String _formatArgs(Map<String, dynamic> args) {
    return args.entries.map((e) => '${e.key}: ${e.value}').join('\n');
  }
}