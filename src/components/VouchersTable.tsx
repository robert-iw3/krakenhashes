import React, { useEffect } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Button,
  Typography,
  Box
} from '@mui/material';
import { format } from 'date-fns';
import { useVouchers } from '../hooks/useVouchers';

export const VouchersTable: React.FC = () => {
  const { vouchers, disableVoucher, fetchVouchers, error } = useVouchers();

  useEffect(() => {
    console.log('Fetching vouchers...');
    fetchVouchers();
  }, [fetchVouchers]);

  useEffect(() => {
    console.log('Current vouchers:', vouchers);
  }, [vouchers]);

  if (error) {
    return (
      <Box sx={{ p: 2 }}>
        <Typography color="error">Error: {error}</Typography>
      </Box>
    );
  }

  if (!vouchers || vouchers.length === 0) {
    return (
      <Box sx={{ p: 2 }}>
        <Typography>No vouchers available.</Typography>
      </Box>
    );
  }

  return (
    <TableContainer component={Paper}>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Voucher Code</TableCell>
            <TableCell>Created By</TableCell>
            <TableCell>Multiple Use</TableCell>
            <TableCell>Created At</TableCell>
            <TableCell>Used By</TableCell>
            <TableCell>Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {Array.isArray(vouchers) && vouchers.map((voucher) => {
            console.log('Rendering voucher:', voucher);
            return (
              <TableRow key={voucher.code}>
                <TableCell>{voucher.code}</TableCell>
                <TableCell>{voucher.created_by?.username || 'Unknown'}</TableCell>
                <TableCell>{voucher.is_continuous ? 'Yes' : 'No'}</TableCell>
                <TableCell>
                  {voucher.created_at ? format(new Date(voucher.created_at), 'yyyy-MM-dd HH:mm:ss') : 'N/A'}
                </TableCell>
                <TableCell>
                  {voucher.used_by_agent_id?.Valid ? `Agent ${voucher.used_by_agent_id.Int64}` : 'Not Used'}
                </TableCell>
                <TableCell>
                  <Button
                    variant="outlined"
                    color="error"
                    onClick={() => disableVoucher(voucher.code)}
                    disabled={!voucher.is_active}
                  >
                    {!voucher.is_active ? 'Disabled' : 'Disable'}
                  </Button>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </TableContainer>
  );
}; 