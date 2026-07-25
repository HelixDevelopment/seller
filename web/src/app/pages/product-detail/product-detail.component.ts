import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink, Router } from '@angular/router';
import { ApiService, Product } from '../../core/api.service';
import { PageHeaderComponent, ConfirmDialogComponent } from '../../shared/index';

@Component({
  selector: 'app-product-detail',
  standalone: true,
  imports: [CommonModule, RouterLink, PageHeaderComponent, ConfirmDialogComponent],
  template: `
    <div class="product-detail">
      <app-page-header [title]="product ? product.name : 'Product Details'">
        <a routerLink="/products" class="back-link">&larr; Back to Products</a>
      </app-page-header>

      <div class="spinner" *ngIf="loading">
        <div class="spinner-icon"></div>
        <span>Loading product...</span>
      </div>

      <div class="error-state" *ngIf="error && !loading">
        <p>{{ error }}</p>
        <a routerLink="/products" class="btn btn-secondary">Back to Products</a>
      </div>

      <ng-container *ngIf="product && !loading">
        <div class="action-buttons">
          <a [routerLink]="['/products', product.id, 'edit']" class="btn btn-secondary">Edit</a>
          <button class="btn btn-danger" (click)="showConfirm = true">Delete</button>
        </div>

        <app-confirm-dialog
          *ngIf="showConfirm"
          title="Delete Product"
          message="Are you sure you want to delete &quot;{{ product.name }}&quot;? This action cannot be undone."
          confirmLabel="Delete"
          (confirm)="deleteProduct()"
          (cancel)="showConfirm = false"
        ></app-confirm-dialog>

        <div class="detail-grid">
          <div class="detail-card">
            <h3>Product Information</h3>
            <dl>
              <dt>Name</dt>
              <dd>{{ product.name }}</dd>
              <dt>Description</dt>
              <dd>{{ product.description || '-' }}</dd>
              <dt>Price</dt>
              <dd>{{ product.price | currency:product.currency }}</dd>
              <dt>Currency</dt>
              <dd>{{ product.currency }}</dd>
            </dl>
          </div>
          <div class="detail-card">
            <h3>Status & Timing</h3>
            <dl>
              <dt>Status</dt>
              <dd>
                <span class="badge" [ngClass]="product.status">{{ product.status }}</span>
              </dd>
              <dt>Created</dt>
              <dd>{{ product.created_at | date:'medium' }}</dd>
              <dt>Updated</dt>
              <dd>{{ product.updated_at | date:'medium' }}</dd>
            </dl>
          </div>
        </div>
      </ng-container>
    </div>
  `,
  styles: [`
    .product-detail { padding: var(--od-spacing-xl); }
    .back-link { color: var(--od-accent); text-decoration: none; font-size: 14px; display: inline-block; margin-bottom: 8px; }
    .back-link:hover { text-decoration: underline; }
    .action-buttons { display: flex; gap: 8px; margin-bottom: 16px; }
    .detail-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
      gap: 16px;
    }
    .detail-card {
      background: var(--od-card-bg);
      border-radius: var(--od-radius);
      padding: 20px;
      box-shadow: var(--od-card-shadow);
    }
    .detail-card h3 { margin: 0 0 16px; font-size: 16px; color: var(--od-text-primary); }
    dl { margin: 0; }
    dt { font-size: 12px; color: var(--od-text-secondary); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 4px; }
    dd { margin: 0 0 16px; font-size: 14px; color: var(--od-text-primary); }
    .badge {
      display: inline-block;
      padding: 2px 10px;
      border-radius: 12px;
      font-size: 12px;
      font-weight: 500;
      text-transform: capitalize;
    }
    .badge.active { background: #dcfce7; color: #166534; }
    .badge.inactive { background: #f3f4f6; color: #6b7280; }
    .badge.archived { background: #fee2e2; color: #991b1b; }
    .spinner {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 12px;
      padding: 48px;
      color: var(--od-text-secondary);
    }
    .spinner-icon {
      width: 20px;
      height: 20px;
      border: 2px solid var(--od-border);
      border-top-color: var(--od-accent);
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }
    @keyframes spin { to { transform: rotate(360deg); } }
    .error-state {
      background: var(--od-bg-danger, #fef2f2);
      border: 1px solid var(--od-border-danger, #fecaca);
      border-radius: var(--od-radius);
      padding: 24px;
      text-align: center;
      color: var(--od-danger, #991b1b);
    }
    .error-state a { margin-top: 12px; display: inline-block; }
    .btn {
      padding: 8px 20px;
      border-radius: var(--od-radius-sm);
      font-size: 14px;
      font-weight: 500;
      cursor: pointer;
      text-decoration: none;
      border: none;
      transition: background-color 0.2s;
    }
    .btn-secondary { background: var(--od-bg-secondary); color: var(--od-text-primary); }
    .btn-secondary:hover { background: var(--od-bg-tertiary); }
    .btn-danger { background: var(--od-danger, #ef4444); color: white; }
    .btn-danger:hover { opacity: 0.9; }
  `]
})
export class ProductDetailComponent implements OnInit {
  product: Product | null = null;
  loading = true;
  error = '';
  showConfirm = false;
  private merchantId = '';

  constructor(
    private route: ActivatedRoute,
    private api: ApiService,
    private router: Router
  ) {}

  ngOnInit(): void {
    this.merchantId = localStorage.getItem('helix_merchant_id') || '';
    const id = this.route.snapshot.paramMap.get('id')!;
    if (!id || !this.merchantId) {
      this.error = 'Product ID or merchant not found.';
      this.loading = false;
      return;
    }
    this.api.getProduct(this.merchantId, id).subscribe({
      next: (data) => {
        this.product = data;
        this.loading = false;
      },
      error: () => {
        this.error = 'Failed to load product.';
        this.loading = false;
      }
    });
  }

  deleteProduct(): void {
    if (!this.product || !this.merchantId) return;
    this.api.deleteProduct(this.merchantId, this.product.id).subscribe({
      next: () => this.router.navigate(['/products']),
      error: () => {
        this.error = 'Failed to delete product. Please try again.';
        this.showConfirm = false;
      }
    });
  }
}
