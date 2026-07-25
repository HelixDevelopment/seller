import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink, Router, ActivatedRoute } from '@angular/router';
import { ApiService, Product } from '../../core/api.service';
import { PageHeaderComponent } from '../../shared/index';

@Component({
  selector: 'app-product-edit',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, PageHeaderComponent],
  template: `
    <div class="product-edit">
      <div class="loading" *ngIf="!loaded">
        <div class="spinner-icon"></div>
        <span>Loading product...</span>
      </div>

      <ng-container *ngIf="loaded">
        <app-page-header title="Edit Product">
          <a routerLink="/products" class="back-link">&larr; Back to Products</a>
        </app-page-header>

        <div class="error-banner" *ngIf="error">
          {{ error }}
        </div>

        <div class="form-card">
          <form (ngSubmit)="onSubmit()" #editForm="ngForm">
            <div class="form-grid">
              <div class="form-group">
                <label for="name">Name *</label>
                <input id="name" type="text" [(ngModel)]="formData.name" name="name" #name="ngModel" required>
                <span class="field-error" *ngIf="name.invalid && name.touched">Name is required</span>
              </div>
              <div class="form-group">
                <label for="price">Price *</label>
                <input id="price" type="number" [(ngModel)]="formData.price" name="price" #price="ngModel" required min="0" pattern="^\\d+(\\.\\d{1,2})?$">
                <span class="field-error" *ngIf="price.invalid && price.touched">Valid price is required</span>
              </div>
              <div class="form-group form-group-full">
                <label for="description">Description</label>
                <textarea id="description" [(ngModel)]="formData.description" name="description" rows="3"></textarea>
              </div>
              <div class="form-group">
                <label for="currency">Currency</label>
                <select id="currency" [(ngModel)]="formData.currency" name="currency">
                  <option value="" disabled>Select currency</option>
                  <option value="USD">USD</option>
                  <option value="EUR">EUR</option>
                  <option value="GBP">GBP</option>
                </select>
              </div>
            </div>
            <div class="form-actions">
              <a routerLink="/products" class="btn btn-secondary">Cancel</a>
              <button type="submit" class="btn btn-primary" [disabled]="submitting || editForm.invalid">
                {{ submitting ? 'Saving...' : 'Save Changes' }}
              </button>
            </div>
          </form>
        </div>
      </ng-container>
    </div>
  `,
  styles: [`
    .product-edit { padding: var(--od-spacing-xl); }
    .back-link { color: var(--od-accent); text-decoration: none; font-size: 14px; display: inline-block; margin-bottom: 8px; }
    .back-link:hover { text-decoration: underline; }
    .loading {
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
    .error-banner {
      background: var(--od-bg-danger, #fef2f2);
      border: 1px solid var(--od-border-danger, #fecaca);
      border-radius: var(--od-radius);
      padding: 12px 16px;
      color: var(--od-danger, #991b1b);
      font-size: 14px;
      margin-bottom: 16px;
    }
    .form-card {
      background: var(--od-card-bg);
      border-radius: var(--od-radius);
      padding: 24px;
      box-shadow: var(--od-card-shadow);
      max-width: 640px;
    }
    .form-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 16px;
    }
    .form-group-full { grid-column: 1 / -1; }
    .form-group { display: flex; flex-direction: column; }
    label { font-size: 13px; font-weight: 500; color: var(--od-text-primary); margin-bottom: 6px; }
    input, select, textarea {
      padding: 8px 12px;
      border: 1px solid var(--od-border);
      border-radius: var(--od-radius-sm);
      font-size: 14px;
      color: var(--od-text-primary);
      background: var(--od-bg-primary);
      outline: none;
      transition: border-color 0.2s;
      font-family: inherit;
    }
    input:focus, select:focus, textarea:focus { border-color: var(--od-accent); }
    input.ng-invalid.ng-touched, select.ng-invalid.ng-touched, textarea.ng-invalid.ng-touched { border-color: var(--od-danger, #ef4444); }
    .field-error { color: var(--od-danger, #ef4444); font-size: 12px; margin-top: 4px; }
    .form-actions {
      display: flex;
      justify-content: flex-end;
      gap: 12px;
      margin-top: 24px;
      padding-top: 16px;
      border-top: 1px solid var(--od-border);
    }
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
    .btn-primary { background: var(--od-accent); color: white; }
    .btn-primary:hover { opacity: 0.9; }
    .btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
    .btn-secondary { background: var(--od-bg-secondary); color: var(--od-text-primary); display: inline-flex; align-items: center; }
    .btn-secondary:hover { background: var(--od-bg-tertiary); }
  `]
})
export class ProductEditComponent implements OnInit {
  formData: Partial<Product> = {
    name: '',
    description: '',
    price: 0,
    currency: 'USD'
  };
  submitting = false;
  loaded = false;
  error = '';
  private merchantId = '';
  private productId = '';

  constructor(private api: ApiService, private router: Router, private route: ActivatedRoute) {}

  ngOnInit(): void {
    this.merchantId = localStorage.getItem('helix_merchant_id') || '';
    this.productId = this.route.snapshot.paramMap.get('id')!;
    if (!this.productId || !this.merchantId) {
      this.router.navigate(['/products']);
      return;
    }
    this.api.getProduct(this.merchantId, this.productId).subscribe({
      next: (product) => {
        this.formData = {
          name: product.name,
          description: product.description || '',
          price: product.price,
          currency: product.currency
        };
        this.loaded = true;
      },
      error: () => this.router.navigate(['/products'])
    });
  }

  onSubmit(): void {
    if (!this.formData.name || !this.formData.price) return;
    this.submitting = true;
    this.error = '';
    this.api.updateProduct(this.merchantId, this.productId, this.formData).subscribe({
      next: () => this.router.navigate(['/products', this.productId]),
      error: () => {
        this.error = 'Failed to update product. Please check your input and try again.';
        this.submitting = false;
      }
    });
  }
}
