import 'package:flutter/material.dart';
import '../models/session.dart';

class SessionCard extends StatefulWidget {
  final Session session;
  final VoidCallback? onTap;
  final Function(String? addAllowed)? onApprove;
  final Function(String? addAllowed)? onReject;

  const SessionCard({
    super.key,
    required this.session,
    this.onTap,
    this.onApprove,
    this.onReject,
  });

  @override
  State<SessionCard> createState() => _SessionCardState();
}

class _SessionCardState extends State<SessionCard> {
  final TextEditingController _addAllowedController = TextEditingController();

  @override
  void dispose() {
    _addAllowedController.dispose();
    super.dispose();
  }

  String _getBlockedInfo() {
    final args = widget.session.blockedArgs;
    if (args == null) return '';

    if (args.containsKey('file_path')) {
      return args['file_path'] as String;
    }
    if (args.containsKey('path')) {
      return args['path'] as String;
    }
    if (args.containsKey('command')) {
      return args['command'] as String;
    }
    return '';
  }

  @override
  Widget build(BuildContext context) {
    final isBlocked = widget.session.state == SessionState.blocked;

    return ConstrainedBox(
      constraints: BoxConstraints(
        minHeight: isBlocked ? 0 : 130,
      ),
      child: Card(
        elevation: 0,
        margin: EdgeInsets.zero,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
          side: BorderSide(
            color: _getBorderColor(),
            width: 2,
          ),
        ),
        child: InkWell(
          onTap: widget.onTap,
          borderRadius: BorderRadius.circular(12),
          child: Padding(
            padding: const EdgeInsets.all(14),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        _getTaskTitle(),
                        style: const TextStyle(
                          fontWeight: FontWeight.bold,
                          fontSize: 14,
                        ),
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    _buildStateBadge(),
                  ],
                ),
                const SizedBox(height: 8),
                Text(
                  widget.session.input.isNotEmpty ? widget.session.input : 'No input',
                  maxLines: 3,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: Colors.grey[700],
                    height: 1.4,
                  ),
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    const Spacer(),
                    if (widget.session.state == SessionState.processing ||
                        widget.session.state == SessionState.planning)
                      _buildRunningIndicator(),
                  ],
                ),
                if (widget.session.state == SessionState.blocked) ...[
                  const SizedBox(height: 12),
                  const Divider(height: 1),
                  const SizedBox(height: 12),
                  _buildBlockedInfoSection(),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _addAllowedController,
                    decoration: InputDecoration(
                      hintText: 'Add allowed path (optional)',
                      hintStyle: TextStyle(color: Colors.grey[400], fontSize: 12),
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                        borderSide: BorderSide(color: Colors.grey[300]!),
                      ),
                      contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                      isDense: true,
                    ),
                    style: const TextStyle(fontSize: 12),
                  ),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      Expanded(
                        child: ElevatedButton.icon(
                          icon: const Icon(Icons.check, size: 16),
                          label: const Text('Approve'),
                          style: ElevatedButton.styleFrom(
                            backgroundColor: Colors.green,
                            foregroundColor: Colors.white,
                            padding: const EdgeInsets.symmetric(vertical: 8),
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(6),
                            ),
                          ),
                          onPressed: widget.onApprove != null
                              ? () => widget.onApprove!(_addAllowedController.text.trim())
                              : null,
                        ),
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: OutlinedButton.icon(
                          icon: const Icon(Icons.close, size: 16),
                          label: const Text('Reject'),
                          style: OutlinedButton.styleFrom(
                            foregroundColor: Colors.red,
                            side: const BorderSide(color: Colors.red),
                            padding: const EdgeInsets.symmetric(vertical: 8),
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(6),
                            ),
                          ),
                          onPressed: widget.onReject != null
                              ? () => widget.onReject!(_addAllowedController.text.trim())
                              : null,
                        ),
                      ),
                    ],
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildBlockedInfoSection() {
    final toolName = widget.session.blockedTool ?? 'Unknown tool';
    final info = _getBlockedInfo();

    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: const Color(0xFFF59E0B).withAlpha(20),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: const Color(0xFFF59E0B).withAlpha(60)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.build, size: 14, color: const Color(0xFFF59E0B)),
              const SizedBox(width: 6),
              Text(
                'Tool: $toolName',
                style: const TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  color: Color(0xFFF59E0B),
                ),
              ),
            ],
          ),
          if (info.isNotEmpty) ...[
            const SizedBox(height: 6),
            Row(
              children: [
                Icon(Icons.folder, size: 14, color: Colors.grey[600]),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    info,
                    style: TextStyle(fontSize: 11, color: Colors.grey[700]),
                    overflow: TextOverflow.ellipsis,
                    maxLines: 2,
                  ),
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }

  Color _getBorderColor() {
    switch (widget.session.state) {
      case SessionState.pending:
        return const Color(0xFF3B82F6);
      case SessionState.planning:
        return const Color(0xFF8B5CF6);
      case SessionState.processing:
        return const Color(0xFF10B981);
      case SessionState.blocked:
        return const Color(0xFFF59E0B);
      case SessionState.completed:
        return const Color(0xFF059669);
    }
  }

  Widget _buildStateBadge() {
    Color color;
    String label;

    switch (widget.session.state) {
      case SessionState.pending:
        color = const Color(0xFF3B82F6);
        label = 'Queued';
        break;
      case SessionState.planning:
        color = const Color(0xFF8B5CF6);
        label = 'Planning';
        break;
      case SessionState.processing:
        color = const Color(0xFF10B981);
        label = 'Active';
        break;
      case SessionState.blocked:
        color = const Color(0xFFF59E0B);
        label = 'Review';
        break;
      case SessionState.completed:
        color = const Color(0xFF059669);
        label = 'Done';
        break;
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: color.withAlpha(30),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(
          color: color.withAlpha(80),
          width: 1,
        ),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: color,
          fontSize: 11,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }

  Widget _buildRunningIndicator() {
    return SizedBox(
      width: 16,
      height: 16,
      child: CircularProgressIndicator(
        strokeWidth: 2,
        valueColor: AlwaysStoppedAnimation<Color>(Colors.blue),
      ),
    );
  }

  String _getTaskTitle() {
    if (widget.session.input.isNotEmpty) {
      final words = widget.session.input.split(' ');
      if (words.length >= 3) {
        return words.take(3).join(' ');
      }
      return widget.session.input;
    }
    return 'Task ${widget.session.id.substring(0, 6)}';
  }
}