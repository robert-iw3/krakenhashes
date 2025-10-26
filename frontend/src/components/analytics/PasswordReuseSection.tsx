/**
 * Password reuse section showing reuse count and list of affected users.
 * Displays one row per password with user lists and hashlist occurrence tracking.
 */
import React, { useState } from 'react';
import {
  Paper,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  Box,
  Chip,
  Button,
  Collapse,
  IconButton,
  Snackbar,
  Alert,
} from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import { ReuseStats, PasswordReuseInfo, UserOccurrence } from '../../types/analytics';
import { threeColumnTableStyles, passwordReuseTableStyles } from './tableStyles';

interface PasswordReuseSectionProps {
  data: ReuseStats;
}

export default function PasswordReuseSection({ data }: PasswordReuseSectionProps) {
  const [page, setPage] = useState(0);
  const [rowsPerPage] = useState(50);
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set());
  const [copySuccess, setCopySuccess] = useState(false);

  const hasData = data.total_reused > 0 && data.password_reuse_info && data.password_reuse_info.length > 0;

  if (!hasData) {
    return null;
  }

  const handleChangePage = (_event: unknown, newPage: number) => {
    setPage(newPage);
  };

  const toggleExpanded = (rowIndex: number) => {
    const newExpanded = new Set(expandedRows);
    if (newExpanded.has(rowIndex)) {
      newExpanded.delete(rowIndex);
    } else {
      newExpanded.add(rowIndex);
    }
    setExpandedRows(newExpanded);
  };

  const formatUsers = (users: UserOccurrence[], rowIndex: number) => {
    const displayLimit = 5;
    const displayUsers = users.slice(0, displayLimit);
    const remainingUsers = users.slice(displayLimit);
    const remainingCount = users.length - displayLimit;

    const formatUserText = (user: UserOccurrence) => `${user.username} (${user.hashlist_count})`;

    const displayText = displayUsers.map(formatUserText).join(', ');

    if (remainingCount > 0) {
      return (
        <Box>
          {displayText}
          <Button
            size="small"
            onClick={() => toggleExpanded(rowIndex)}
            endIcon={expandedRows.has(rowIndex) ? <ExpandLessIcon /> : <ExpandMoreIcon />}
            sx={{ ml: 1 }}
          >
            ...and {remainingCount} more
          </Button>
          <Collapse in={expandedRows.has(rowIndex)}>
            <Box sx={{ mt: 1, pl: 2 }}>
              {remainingUsers.map((user, idx) => (
                <Typography key={idx} variant="body2" sx={{ py: 0.5 }}>
                  {formatUserText(user)}
                </Typography>
              ))}
            </Box>
          </Collapse>
        </Box>
      );
    }

    return displayText;
  };

  const copyUsernames = (users: UserOccurrence[]) => {
    const usernames = users.map(u => u.username).join(', ');
    navigator.clipboard.writeText(usernames);
    setCopySuccess(true);
  };

  const handleCloseSnackbar = () => {
    setCopySuccess(false);
  };

  // Pagination
  const paginatedData = data.password_reuse_info.slice(
    page * rowsPerPage,
    page * rowsPerPage + rowsPerPage
  );

  return (
    <Paper sx={{ p: 3, mb: 3 }}>
      <Typography variant="h5" gutterBottom>
        Password Reuse
      </Typography>

      {/* Summary */}
      <Box sx={{ mb: 3 }}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell sx={threeColumnTableStyles.labelCell}>Metric</TableCell>
              <TableCell sx={threeColumnTableStyles.countCell}>Count</TableCell>
              <TableCell sx={threeColumnTableStyles.percentageCell}>Percentage</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            <TableRow>
              <TableCell sx={threeColumnTableStyles.labelCell}>Passwords Reused</TableCell>
              <TableCell sx={threeColumnTableStyles.countCell}>{data.total_reused.toLocaleString()}</TableCell>
              <TableCell sx={threeColumnTableStyles.percentageCell}>{data.percentage_reused.toFixed(2)}%</TableCell>
            </TableRow>
            <TableRow>
              <TableCell sx={threeColumnTableStyles.labelCell}>Unique Passwords</TableCell>
              <TableCell sx={threeColumnTableStyles.countCell}>{data.total_unique.toLocaleString()}</TableCell>
              <TableCell sx={threeColumnTableStyles.percentageCell}>{(100 - data.percentage_reused).toFixed(2)}%</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </Box>

      {/* Password Reuse Table */}
      <Box>
        <Typography variant="h6" gutterBottom>
          Reused Passwords by Occurrence
        </Typography>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell sx={passwordReuseTableStyles.passwordCell}>Password</TableCell>
                <TableCell sx={passwordReuseTableStyles.usersCell}>Users (Hashlist Count)</TableCell>
                <TableCell sx={passwordReuseTableStyles.occurrencesCell}>Total Occurrences</TableCell>
                <TableCell sx={passwordReuseTableStyles.userCountCell}>User Count</TableCell>
                <TableCell sx={passwordReuseTableStyles.actionsCell}>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {paginatedData.map((passwordInfo: PasswordReuseInfo, index) => {
                const globalIndex = page * rowsPerPage + index;
                return (
                  <TableRow key={index}>
                    <TableCell sx={passwordReuseTableStyles.passwordCell}>
                      <Chip label={passwordInfo.password} size="small" />
                    </TableCell>
                    <TableCell sx={passwordReuseTableStyles.usersCell}>{formatUsers(passwordInfo.users, globalIndex)}</TableCell>
                    <TableCell sx={passwordReuseTableStyles.occurrencesCell}>{passwordInfo.total_occurrences}</TableCell>
                    <TableCell sx={passwordReuseTableStyles.userCountCell}>{passwordInfo.user_count}</TableCell>
                    <TableCell sx={passwordReuseTableStyles.actionsCell}>
                      <IconButton
                        size="small"
                        onClick={() => copyUsernames(passwordInfo.users)}
                        title="Copy usernames to clipboard"
                      >
                        <ContentCopyIcon fontSize="small" />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>
        <TablePagination
          rowsPerPageOptions={[50]}
          component="div"
          count={data.password_reuse_info.length}
          rowsPerPage={rowsPerPage}
          page={page}
          onPageChange={handleChangePage}
        />
      </Box>

      {/* Success Snackbar */}
      <Snackbar
        open={copySuccess}
        autoHideDuration={3000}
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={handleCloseSnackbar} severity="success" sx={{ width: '100%' }}>
          Usernames copied to clipboard!
        </Alert>
      </Snackbar>
    </Paper>
  );
}
